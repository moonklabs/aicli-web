package terminal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

// Serializer 터미널 직렬화기
type Serializer struct {
	config *SerializerConfig
}

// SerializerConfig 직렬화기 설정
type SerializerConfig struct {
	Format           SerializationFormat
	EnableCompression bool
	CompressionLevel int
	PrettyPrint      bool
}

// SerializationFormat 직렬화 형식
type SerializationFormat int

const (
	FormatJSON SerializationFormat = iota
	FormatBinary
	FormatText
)

// DefaultSerializerConfig 기본 직렬화기 설정
func DefaultSerializerConfig() *SerializerConfig {
	return &SerializerConfig{
		Format:           FormatJSON,
		EnableCompression: true,
		CompressionLevel: gzip.BestSpeed,
		PrettyPrint:      false,
	}
}

// NewSerializer 새 직렬화기 생성
func NewSerializer(config *SerializerConfig) *Serializer {
	if config == nil {
		config = DefaultSerializerConfig()
	}
	
	return &Serializer{
		config: config,
	}
}

// SerializeSnapshot 스냅샷 직렬화
func (s *Serializer) SerializeSnapshot(snapshot *Snapshot) ([]byte, error) {
	switch s.config.Format {
	case FormatJSON:
		return s.serializeJSON(snapshot)
	case FormatBinary:
		return s.serializeBinary(snapshot)
	case FormatText:
		return s.serializeText(snapshot)
	default:
		return nil, fmt.Errorf("unsupported format: %v", s.config.Format)
	}
}

// DeserializeSnapshot 스냅샷 역직렬화
func (s *Serializer) DeserializeSnapshot(data []byte) (*Snapshot, error) {
	// 압축 해제
	if s.config.EnableCompression {
		decompressed, err := s.decompress(data)
		if err == nil {
			data = decompressed
		}
	}
	
	switch s.config.Format {
	case FormatJSON:
		return s.deserializeJSON(data)
	case FormatBinary:
		return s.deserializeBinary(data)
	case FormatText:
		return s.deserializeText(data)
	default:
		return nil, fmt.Errorf("unsupported format: %v", s.config.Format)
	}
}

// SerializeScreen 화면 직렬화
func (s *Serializer) SerializeScreen(screen *Screen) ([]byte, error) {
	data, err := json.Marshal(screen)
	if err != nil {
		return nil, err
	}
	
	if s.config.EnableCompression {
		return s.compress(data)
	}
	
	return data, nil
}

// DeserializeScreen 화면 역직렬화
func (s *Serializer) DeserializeScreen(data []byte) (*Screen, error) {
	// 압축 해제
	if s.config.EnableCompression {
		decompressed, err := s.decompress(data)
		if err == nil {
			data = decompressed
		}
	}
	
	var screen Screen
	if err := json.Unmarshal(data, &screen); err != nil {
		return nil, err
	}
	
	return &screen, nil
}

// serializeJSON JSON 직렬화
func (s *Serializer) serializeJSON(snapshot *Snapshot) ([]byte, error) {
	var data []byte
	var err error
	
	if s.config.PrettyPrint {
		data, err = json.MarshalIndent(snapshot, "", "  ")
	} else {
		data, err = json.Marshal(snapshot)
	}
	
	if err != nil {
		return nil, err
	}
	
	if s.config.EnableCompression {
		return s.compress(data)
	}
	
	return data, nil
}

// deserializeJSON JSON 역직렬화
func (s *Serializer) deserializeJSON(data []byte) (*Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	
	return &snapshot, nil
}

// serializeBinary 바이너리 직렬화
func (s *Serializer) serializeBinary(snapshot *Snapshot) ([]byte, error) {
	// 간단한 바이너리 형식 구현
	var buf bytes.Buffer
	
	// 헤더 쓰기
	header := BinaryHeader{
		Magic:   0x534E4150, // "SNAP"
		Version: 1,
		Flags:   0,
	}
	
	if s.config.EnableCompression {
		header.Flags |= FlagCompressed
	}
	
	if err := writeBinaryHeader(&buf, header); err != nil {
		return nil, err
	}
	
	// 데이터 쓰기
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	
	if s.config.EnableCompression {
		data, err = s.compress(data)
		if err != nil {
			return nil, err
		}
	}
	
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// deserializeBinary 바이너리 역직렬화
func (s *Serializer) deserializeBinary(data []byte) (*Snapshot, error) {
	buf := bytes.NewReader(data)
	
	// 헤더 읽기
	header, err := readBinaryHeader(buf)
	if err != nil {
		return nil, err
	}
	
	if header.Magic != 0x534E4150 {
		return nil, fmt.Errorf("invalid magic number: %x", header.Magic)
	}
	
	// 데이터 읽기
	payload := make([]byte, buf.Len())
	if _, err := buf.Read(payload); err != nil {
		return nil, err
	}
	
	// 압축 해제
	if header.Flags&FlagCompressed != 0 {
		payload, err = s.decompress(payload)
		if err != nil {
			return nil, err
		}
	}
	
	// JSON 역직렬화
	var snapshot Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, err
	}
	
	return &snapshot, nil
}

// serializeText 텍스트 직렬화
func (s *Serializer) serializeText(snapshot *Snapshot) ([]byte, error) {
	var buf bytes.Buffer
	
	// 헤더 정보
	fmt.Fprintf(&buf, "=== Terminal Snapshot ===\n")
	fmt.Fprintf(&buf, "ID: %s\n", snapshot.ID)
	fmt.Fprintf(&buf, "Session: %s\n", snapshot.SessionID)
	fmt.Fprintf(&buf, "Timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(&buf, "Size: %dx%d\n", snapshot.Screen.Rows, snapshot.Screen.Cols)
	fmt.Fprintf(&buf, "\n")
	
	// 화면 내용
	fmt.Fprintf(&buf, "=== Screen Content ===\n")
	for i, line := range snapshot.Screen.Lines {
		fmt.Fprintf(&buf, "%3d: ", i+1)
		for _, cell := range line.Cells {
			if cell.Rune == 0 {
				buf.WriteRune(' ')
			} else {
				buf.WriteRune(cell.Rune)
			}
		}
		buf.WriteRune('\n')
	}
	
	// 스크롤백
	if len(snapshot.ScrollBack) > 0 {
		fmt.Fprintf(&buf, "\n=== ScrollBack (%d lines) ===\n", len(snapshot.ScrollBack))
		for _, line := range snapshot.ScrollBack {
			fmt.Fprintf(&buf, "%s\n", line)
		}
	}
	
	data := buf.Bytes()
	
	if s.config.EnableCompression {
		return s.compress(data)
	}
	
	return data, nil
}

// deserializeText 텍스트 역직렬화
func (s *Serializer) deserializeText(data []byte) (*Snapshot, error) {
	// 텍스트 형식은 읽기 전용
	return nil, fmt.Errorf("text format is read-only")
}

// compress 압축
func (s *Serializer) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	
	gw, err := gzip.NewWriterLevel(&buf, s.config.CompressionLevel)
	if err != nil {
		return nil, err
	}
	
	if _, err := gw.Write(data); err != nil {
		gw.Close()
		return nil, err
	}
	
	if err := gw.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompress 압축 해제
func (s *Serializer) decompress(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, gr); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// ExportSnapshot 스냅샷 내보내기
func (s *Serializer) ExportSnapshot(snapshot *Snapshot, format ExportFormat) (string, error) {
	switch format {
	case ExportFormatBase64:
		return s.exportBase64(snapshot)
	case ExportFormatHex:
		return s.exportHex(snapshot)
	case ExportFormatURL:
		return s.exportURL(snapshot)
	default:
		return "", fmt.Errorf("unsupported export format: %v", format)
	}
}

// ImportSnapshot 스냅샷 가져오기
func (s *Serializer) ImportSnapshot(data string, format ExportFormat) (*Snapshot, error) {
	var rawData []byte
	var err error
	
	switch format {
	case ExportFormatBase64:
		rawData, err = s.importBase64(data)
	case ExportFormatHex:
		rawData, err = s.importHex(data)
	case ExportFormatURL:
		rawData, err = s.importURL(data)
	default:
		return nil, fmt.Errorf("unsupported import format: %v", format)
	}
	
	if err != nil {
		return nil, err
	}
	
	return s.DeserializeSnapshot(rawData)
}

// exportBase64 Base64로 내보내기
func (s *Serializer) exportBase64(snapshot *Snapshot) (string, error) {
	data, err := s.SerializeSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(data), nil
}

// importBase64 Base64에서 가져오기
func (s *Serializer) importBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// exportHex 16진수로 내보내기
func (s *Serializer) exportHex(snapshot *Snapshot) (string, error) {
	data, err := s.SerializeSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", data), nil
}

// importHex 16진수에서 가져오기
func (s *Serializer) importHex(hex string) ([]byte, error) {
	data := make([]byte, len(hex)/2)
	
	for i := 0; i < len(hex); i += 2 {
		var b byte
		fmt.Sscanf(hex[i:i+2], "%02x", &b)
		data[i/2] = b
	}
	
	return data, nil
}

// exportURL URL 형식으로 내보내기
func (s *Serializer) exportURL(snapshot *Snapshot) (string, error) {
	base64Data, err := s.exportBase64(snapshot)
	if err != nil {
		return "", err
	}
	
	// URL 안전 문자로 변환
	urlSafe := base64.URLEncoding.EncodeToString([]byte(base64Data))
	
	return fmt.Sprintf("snapshot://%s", urlSafe), nil
}

// importURL URL 형식에서 가져오기
func (s *Serializer) importURL(url string) ([]byte, error) {
	if len(url) < 11 || url[:11] != "snapshot://" {
		return nil, fmt.Errorf("invalid snapshot URL")
	}
	
	urlSafe := url[11:]
	base64Data, err := base64.URLEncoding.DecodeString(urlSafe)
	if err != nil {
		return nil, err
	}
	
	return s.importBase64(string(base64Data))
}

// BinaryHeader 바이너리 헤더
type BinaryHeader struct {
	Magic   uint32
	Version uint16
	Flags   uint16
}

// 플래그
const (
	FlagCompressed uint16 = 1 << iota
	FlagEncrypted
)

// writeBinaryHeader 바이너리 헤더 쓰기
func writeBinaryHeader(w io.Writer, h BinaryHeader) error {
	data := make([]byte, 8)
	
	// Big-endian 인코딩
	data[0] = byte(h.Magic >> 24)
	data[1] = byte(h.Magic >> 16)
	data[2] = byte(h.Magic >> 8)
	data[3] = byte(h.Magic)
	data[4] = byte(h.Version >> 8)
	data[5] = byte(h.Version)
	data[6] = byte(h.Flags >> 8)
	data[7] = byte(h.Flags)
	
	_, err := w.Write(data)
	return err
}

// readBinaryHeader 바이너리 헤더 읽기
func readBinaryHeader(r io.Reader) (BinaryHeader, error) {
	data := make([]byte, 8)
	
	if _, err := io.ReadFull(r, data); err != nil {
		return BinaryHeader{}, err
	}
	
	// Big-endian 디코딩
	h := BinaryHeader{
		Magic:   uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]),
		Version: uint16(data[4])<<8 | uint16(data[5]),
		Flags:   uint16(data[6])<<8 | uint16(data[7]),
	}
	
	return h, nil
}

// ExportFormat 내보내기 형식
type ExportFormat int

const (
	ExportFormatBase64 ExportFormat = iota
	ExportFormatHex
	ExportFormatURL
)

// GetStats 통계 조회
func (s *Serializer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"format":            s.config.Format,
		"compression":       s.config.EnableCompression,
		"compression_level": s.config.CompressionLevel,
		"pretty_print":      s.config.PrettyPrint,
	}
}