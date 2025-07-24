package storage

// New 기본 스토리지 인스턴스를 생성합니다 (메모리 스토리지 사용)
// 이 함수는 CLI에서 간편하게 사용하기 위한 헬퍼 함수입니다
//
// 주의: 이 함수는 임시 해결책입니다. 추후 storage 인터페이스와 
// memory 구현체 간의 호환성 문제를 해결해야 합니다.
func New() (Storage, error) {
	// 현재는 memory 구현체가 Storage 인터페이스와 완전히 호환되지 않으므로
	// 임시로 nil을 반환합니다. CLI는 직접 memory.New()를 사용해야 합니다.
	return nil, nil
}

// NewWithConfig 설정에 따른 스토리지 인스턴스를 생성합니다
func NewWithConfig(config StorageConfig) (Storage, error) {
	factory := NewDefaultStorageFactory()
	return factory.Create(config)
}