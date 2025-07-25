package version

import (
	"encoding/json"
	"testing"
)

func BenchmarkGet(b *testing.B) {
	// Get 함수 성능 측정
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Get()
	}
}

func BenchmarkGetParallel(b *testing.B) {
	// Get 함수 병렬 성능 측정
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Get()
		}
	})
}

func BenchmarkVersionInfoJSON(b *testing.B) {
	// VersionInfo JSON 마샬링 성능 측정
	info := Get()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVersionInfoString(b *testing.B) {
	// 버전 정보 문자열 생성 성능 측정
	info := Get()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = info.Version + " " + info.GitCommit + " " + info.BuildTime
	}
}