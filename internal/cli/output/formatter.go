package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/viper"
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
	// Print는 데이터를 지정된 형식으로 출력합니다.
	Print(data interface{}) error
	// PrintTo는 데이터를 지정된 Writer로 출력합니다.
	PrintTo(w io.Writer, data interface{}) error
}

// formatter는 기본 포맷터 구현체입니다.
type formatter struct {
	format Format
}

// New는 지정된 포맷의 새 Formatter를 생성합니다.
func New(format Format) Formatter {
	return &formatter{format: format}
}

// Default는 설정에서 지정된 기본 포맷의 Formatter를 생성합니다.
func Default() Formatter {
	formatStr := viper.GetString("output")
	if formatStr == "" {
		formatStr = string(FormatTable)
	}
	return New(Format(formatStr))
}

// Print는 데이터를 표준 출력으로 출력합니다.
func (f *formatter) Print(data interface{}) error {
	return f.PrintTo(os.Stdout, data)
}

// PrintTo는 데이터를 지정된 Writer로 출력합니다.
func (f *formatter) PrintTo(w io.Writer, data interface{}) error {
	switch f.format {
	case FormatJSON:
		return f.printJSON(w, data)
	case FormatYAML:
		return f.printYAML(w, data)
	case FormatTable:
		return f.printTable(w, data)
	default:
		return fmt.Errorf("지원하지 않는 출력 형식: %s", f.format)
	}
}

// printJSON은 데이터를 JSON 형식으로 출력합니다.
func (f *formatter) printJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML은 데이터를 YAML 형식으로 출력합니다.
func (f *formatter) printYAML(w io.Writer, data interface{}) error {
	// TODO: YAML 라이브러리 추가 후 구현
	// 임시로 JSON 출력
	fmt.Fprintln(w, "# YAML output not yet implemented, showing JSON:")
	return f.printJSON(w, data)
}

// printTable은 데이터를 테이블 형식으로 출력합니다.
func (f *formatter) printTable(w io.Writer, data interface{}) error {
	// 데이터 타입에 따라 다른 테이블 형식 적용
	switch v := data.(type) {
	case []map[string]interface{}:
		return f.printMapSliceTable(w, v)
	case map[string]interface{}:
		return f.printMapTable(w, v)
	default:
		// 기본적으로 문자열로 출력
		fmt.Fprintln(w, data)
		return nil
	}
}

// printMapSliceTable은 맵 슬라이스를 테이블로 출력합니다.
func (f *formatter) printMapSliceTable(w io.Writer, data []map[string]interface{}) error {
	if len(data) == 0 {
		fmt.Fprintln(w, "No data to display.")
		return nil
	}

	// 헤더 추출
	headers := make([]string, 0)
	for key := range data[0] {
		headers = append(headers, key)
	}

	// 간단한 테이블 출력 (추후 더 나은 테이블 라이브러리로 교체)
	// 헤더 출력
	for i, header := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, header)
	}
	fmt.Fprintln(w)

	// 구분선
	for i := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, "-----")
	}
	fmt.Fprintln(w)

	// 데이터 출력
	for _, row := range data {
		for i, header := range headers {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, row[header])
		}
		fmt.Fprintln(w)
	}

	return nil
}

// printMapTable은 단일 맵을 테이블로 출력합니다.
func (f *formatter) printMapTable(w io.Writer, data map[string]interface{}) error {
	// 키-값 형식으로 출력
	for key, value := range data {
		fmt.Fprintf(w, "%s:\t%v\n", key, value)
	}
	return nil
}