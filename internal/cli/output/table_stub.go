package output

import (
	"io"
	"strings"
)

// Table는 간단한 테이블 구현체입니다.
type Table struct {
	writer  io.Writer
	headers []string
	rows    [][]string
}

// NewWriter는 새로운 테이블 라이터를 생성합니다.
func NewWriter(writer io.Writer) *Table {
	return &Table{
		writer: writer,
		rows:   make([][]string, 0),
	}
}

// SetHeader는 테이블 헤더를 설정합니다.
func (t *Table) SetHeader(headers []string) {
	t.headers = headers
}

// Append는 테이블에 행을 추가합니다.
func (t *Table) Append(row []string) {
	t.rows = append(t.rows, row)
}

// Render는 테이블을 렌더링합니다.
func (t *Table) Render() {
	if len(t.headers) > 0 {
		t.writer.Write([]byte(strings.Join(t.headers, "\t") + "\n"))
		t.writer.Write([]byte(strings.Repeat("-", len(strings.Join(t.headers, "\t"))) + "\n"))
	}
	
	for _, row := range t.rows {
		t.writer.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
}

// SetBorder, SetAlignment 등은 호환성을 위한 스텁 메서드들
func (t *Table) SetBorder(bool) {}
func (t *Table) SetAlignment(int) {}
func (t *Table) SetHeaderAlignment(int) {}
func (t *Table) SetHeaderColor(...interface{}) {}
func (t *Table) SetBorders(interface{}) {}
func (t *Table) SetCenterSeparator(string) {}

// 상수들
const (
	ALIGN_LEFT = 0
)

type Border struct {
	Left, Top, Right, Bottom bool
}

type Colors []interface{}

const (
	Bold = iota
	FgCyanColor
)