package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Format은 출력 포맷 타입입니다.
type Format string

const (
	// FormatTable은 테이블 형식 출력입니다.
	FormatTable Format = "table"
	// FormatJSON은 JSON 형식 출력입니다.
	FormatJSON Format = "json"
	// FormatYAML은 YAML 형식 출력입니다.
	FormatYAML Format = "yaml"
)

// Formatter는 출력 포맷터 인터페이스입니다.
type Formatter interface {
	// Format은 데이터를 지정된 형식으로 포맷합니다.
	Format(data interface{}) (string, error)
	// SetHeaders는 테이블 헤더를 설정합니다.
	SetHeaders(headers []string)
	// SetColorEnabled는 색상 출력을 활성화/비활성화합니다.
	SetColorEnabled(enabled bool)
}

// baseFormatter는 모든 포맷터의 기본 구조체입니다.
type baseFormatter struct {
	headers      []string
	colorEnabled bool
}

// TableFormatter는 테이블 형식 출력을 담당합니다.
type TableFormatter struct {
	baseFormatter
}

// JSONFormatter는 JSON 형식 출력을 담당합니다.
type JSONFormatter struct {
	baseFormatter
	indent bool
}

// YAMLFormatter는 YAML 형식 출력을 담당합니다.
type YAMLFormatter struct {
	baseFormatter
}

// FormatterManager는 포맷터를 관리하는 구조체입니다.
type FormatterManager struct {
	format       Format
	formatter    Formatter
	colorEnabled bool
}

// NewFormatterManager는 새로운 FormatterManager를 생성합니다.
func NewFormatterManager(format Format) *FormatterManager {
	fm := &FormatterManager{
		format:       format,
		colorEnabled: detectColorSupport(),
	}
	
	// 포맷에 따라 적절한 포맷터 생성
	switch format {
	case FormatJSON:
		fm.formatter = &JSONFormatter{
			baseFormatter: baseFormatter{colorEnabled: fm.colorEnabled},
			indent:        true,
		}
	case FormatYAML:
		fm.formatter = &YAMLFormatter{
			baseFormatter: baseFormatter{colorEnabled: fm.colorEnabled},
		}
	case FormatTable:
		fallthrough
	default:
		fm.formatter = &TableFormatter{
			baseFormatter: baseFormatter{colorEnabled: fm.colorEnabled},
		}
	}
	
	return fm
}

// DefaultFormatterManager는 설정에서 지정된 기본 포맷의 FormatterManager를 생성합니다.
func DefaultFormatterManager() *FormatterManager {
	formatStr := viper.GetString("output")
	if formatStr == "" {
		formatStr = string(FormatTable)
	}
	return NewFormatterManager(Format(formatStr))
}

// Print는 데이터를 표준 출력으로 출력합니다.
func (fm *FormatterManager) Print(data interface{}) error {
	return fm.PrintTo(os.Stdout, data)
}

// PrintTo는 데이터를 지정된 Writer로 출력합니다.
func (fm *FormatterManager) PrintTo(w io.Writer, data interface{}) error {
	output, err := fm.formatter.Format(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, output)
	return err
}

// SetHeaders는 테이블 헤더를 설정합니다.
func (fm *FormatterManager) SetHeaders(headers []string) {
	fm.formatter.SetHeaders(headers)
}

// SetColorEnabled는 색상 출력을 활성화/비활성화합니다.
func (fm *FormatterManager) SetColorEnabled(enabled bool) {
	fm.colorEnabled = enabled
	fm.formatter.SetColorEnabled(enabled)
}

// detectColorSupport는 터미널의 색상 지원 여부를 감지합니다.
func detectColorSupport() bool {
	// NO_COLOR 환경 변수 확인
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	
	// 터미널인지 확인
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}
	
	// TERM 환경 변수 확인
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}
	
	return true
}

// SetHeaders는 테이블 헤더를 설정합니다.
func (f *baseFormatter) SetHeaders(headers []string) {
	f.headers = headers
}

// SetColorEnabled는 색상 출력을 활성화/비활성화합니다.
func (f *baseFormatter) SetColorEnabled(enabled bool) {
	f.colorEnabled = enabled
}

// Format은 데이터를 테이블 형식으로 포맷합니다.
func (tf *TableFormatter) Format(data interface{}) (string, error) {
	var output strings.Builder
	table := tablewriter.NewWriter(&output)
	
	// 테이블 스타일 설정
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	
	// 색상 설정
	if tf.colorEnabled {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
	}
	
	// 데이터 타입에 따라 처리
	switch v := data.(type) {
	case []map[string]interface{}:
		if len(v) == 0 {
			return "No data to display.\n", nil
		}
		
		// 헤더 설정
		if len(tf.headers) == 0 {
			// 헤더가 설정되지 않은 경우 첫 번째 항목에서 추출
			for key := range v[0] {
				tf.headers = append(tf.headers, key)
			}
		}
		table.SetHeader(tf.headers)
		
		// 데이터 추가
		for _, row := range v {
			rowData := make([]string, len(tf.headers))
			for i, header := range tf.headers {
				if val, ok := row[header]; ok {
					rowData[i] = fmt.Sprintf("%v", val)
				} else {
					rowData[i] = ""
				}
			}
			table.Append(rowData)
		}
		
	case map[string]interface{}:
		// 키-값 형식의 2열 테이블
		table.SetHeader([]string{"Key", "Value"})
		for key, value := range v {
			table.Append([]string{key, fmt.Sprintf("%v", value)})
		}
		
	case [][]string:
		// 이미 문자열 슬라이스 형태인 경우
		if len(tf.headers) > 0 {
			table.SetHeader(tf.headers)
		}
		for _, row := range v {
			table.Append(row)
		}
		
	default:
		// 리플렉션을 사용하여 구조체 슬라이스 처리
		return tf.formatStructSlice(data)
	}
	
	table.Render()
	return output.String(), nil
}

// formatStructSlice는 구조체 슬라이스를 테이블로 포맷합니다.
func (tf *TableFormatter) formatStructSlice(data interface{}) (string, error) {
	value := reflect.ValueOf(data)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Slice {
		// 단일 객체인 경우
		return tf.formatSingleStruct(data)
	}
	
	if value.Len() == 0 {
		return "No data to display.\n", nil
	}
	
	var output strings.Builder
	table := tablewriter.NewWriter(&output)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	
	// 첫 번째 요소에서 필드 이름 추출
	firstElem := value.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}
	
	if firstElem.Kind() != reflect.Struct {
		// 구조체가 아닌 경우 기본 문자열 출력
		return fmt.Sprintf("%v\n", data), nil
	}
	
	// 헤더 설정
	elemType := firstElem.Type()
	headers := make([]string, 0, elemType.NumField())
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		// 태그나 필드 이름 사용
		tag := field.Tag.Get("json")
		if tag != "" && tag != "-" {
			tag = strings.Split(tag, ",")[0]
			headers = append(headers, tag)
		} else if field.IsExported() {
			headers = append(headers, field.Name)
		}
	}
	
	if tf.colorEnabled {
		colorHeaders := make([]tablewriter.Colors, len(headers))
		for i := range colorHeaders {
			colorHeaders[i] = tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor}
		}
		table.SetHeaderColor(colorHeaders...)
	}
	
	table.SetHeader(headers)
	
	// 데이터 추가
	for i := 0; i < value.Len(); i++ {
		elem := value.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		
		row := make([]string, 0, len(headers))
		for j := 0; j < elemType.NumField(); j++ {
			field := elemType.Field(j)
			tag := field.Tag.Get("json")
			if (tag == "-") || (!field.IsExported() && tag == "") {
				continue
			}
			
			fieldValue := elem.Field(j)
			row = append(row, fmt.Sprintf("%v", fieldValue.Interface()))
		}
		table.Append(row)
	}
	
	table.Render()
	return output.String(), nil
}

// formatSingleStruct는 단일 구조체를 키-값 테이블로 포맷합니다.
func (tf *TableFormatter) formatSingleStruct(data interface{}) (string, error) {
	value := reflect.ValueOf(data)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Struct {
		return fmt.Sprintf("%v\n", data), nil
	}
	
	var output strings.Builder
	table := tablewriter.NewWriter(&output)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	
	if tf.colorEnabled {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
	}
	
	table.SetHeader([]string{"Field", "Value"})
	
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		tag := field.Tag.Get("json")
		
		if tag == "-" || (!field.IsExported() && tag == "") {
			continue
		}
		
		fieldName := field.Name
		if tag != "" {
			fieldName = strings.Split(tag, ",")[0]
		}
		
		fieldValue := value.Field(i)
		table.Append([]string{fieldName, fmt.Sprintf("%v", fieldValue.Interface())})
	}
	
	table.Render()
	return output.String(), nil
}

// Format은 데이터를 JSON 형식으로 포맷합니다.
func (jf *JSONFormatter) Format(data interface{}) (string, error) {
	var output strings.Builder
	encoder := json.NewEncoder(&output)
	
	if jf.indent {
		encoder.SetIndent("", "  ")
	}
	
	if err := encoder.Encode(data); err != nil {
		return "", err
	}
	
	result := output.String()
	
	// 색상 지원이 활성화된 경우 문법 하이라이팅 적용
	if jf.colorEnabled {
		result = jf.highlightJSON(result)
	}
	
	return result, nil
}

// highlightJSON은 JSON 문자열에 색상을 적용합니다.
func (jf *JSONFormatter) highlightJSON(jsonStr string) string {
	// 간단한 JSON 문법 하이라이팅
	// 실제 구현에서는 더 정교한 파서를 사용할 수 있습니다.
	lines := strings.Split(jsonStr, "\n")
	for i, line := range lines {
		// 키 하이라이팅
		if strings.Contains(line, `"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && strings.Contains(parts[0], `"`) {
				// 키 부분에 색상 적용
				keyStart := strings.Index(parts[0], `"`)
				keyEnd := strings.LastIndex(parts[0], `"`)
				if keyStart != -1 && keyEnd != -1 && keyStart < keyEnd {
					key := parts[0][keyStart : keyEnd+1]
					coloredKey := color.CyanString(key)
					parts[0] = strings.Replace(parts[0], key, coloredKey, 1)
					lines[i] = parts[0] + ":" + parts[1]
				}
			}
		}
		
		// 숫자 하이라이팅
		// 문자열 하이라이팅 등 추가 가능
	}
	
	return strings.Join(lines, "\n")
}

// Format은 데이터를 YAML 형식으로 포맷합니다.
func (yf *YAMLFormatter) Format(data interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	
	result := string(yamlBytes)
	
	// 색상 지원이 활성화된 경우 문법 하이라이팅 적용
	if yf.colorEnabled {
		result = yf.highlightYAML(result)
	}
	
	return result, nil
}

// highlightYAML은 YAML 문자열에 색상을 적용합니다.
func (yf *YAMLFormatter) highlightYAML(yamlStr string) string {
	// 간단한 YAML 문법 하이라이팅
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		// 키 하이라이팅
		if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// 들여쓰기 유지
				indent := len(line) - len(strings.TrimLeft(line, " "))
				key := strings.TrimSpace(parts[0])
				coloredKey := color.CyanString(key)
				lines[i] = strings.Repeat(" ", indent) + coloredKey + ":" + parts[1]
			}
		}
		
		// 주석 하이라이팅
		if strings.Contains(line, "#") {
			commentStart := strings.Index(line, "#")
			if commentStart != -1 {
				comment := line[commentStart:]
				coloredComment := color.HiBlackString(comment)
				lines[i] = line[:commentStart] + coloredComment
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

// ValidateFormat은 주어진 문자열이 유효한 출력 형식인지 확인합니다.
func ValidateFormat(format string) error {
	switch Format(format) {
	case FormatTable, FormatJSON, FormatYAML:
		return nil
	default:
		return fmt.Errorf("지원하지 않는 출력 형식: %s (지원: table, json, yaml)", format)
	}
}

// GetFormatFromString은 문자열을 Format 타입으로 변환합니다.
func GetFormatFromString(format string) Format {
	switch strings.ToLower(format) {
	case "json":
		return FormatJSON
	case "yaml", "yml":
		return FormatYAML
	case "table":
		return FormatTable
	default:
		return FormatTable
	}
}