package config

import (
	"sync"
)

var (
	// 싱글톤 ConfigManager 인스턴스
	instance *ConfigManager
	once     sync.Once
	initErr  error
)

// GetManager는 싱글톤 ConfigManager 인스턴스를 반환합니다
func GetManager() (*ConfigManager, error) {
	once.Do(func() {
		instance, initErr = NewConfigManager()
		if initErr == nil {
			// 설정 파일 변경 감지 시작
			initErr = instance.Watch()
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return instance, nil
}

// MustGetManager는 싱글톤 ConfigManager 인스턴스를 반환하거나 패닉합니다
func MustGetManager() *ConfigManager {
	cm, err := GetManager()
	if err != nil {
		panic(err)
	}
	return cm
}

// Get은 전역 설정 값을 가져옵니다
func Get(key string) interface{} {
	cm := MustGetManager()
	return cm.Get(key)
}

// GetString은 전역 문자열 설정 값을 가져옵니다
func GetString(key string) string {
	cm := MustGetManager()
	return cm.GetString(key)
}

// GetInt는 전역 정수 설정 값을 가져옵니다
func GetInt(key string) int {
	cm := MustGetManager()
	return cm.GetInt(key)
}

// GetFloat64는 전역 실수 설정 값을 가져옵니다
func GetFloat64(key string) float64 {
	cm := MustGetManager()
	return cm.GetFloat64(key)
}

// GetBool은 전역 불린 설정 값을 가져옵니다
func GetBool(key string) bool {
	cm := MustGetManager()
	return cm.GetBool(key)
}

// Set은 전역 설정 값을 설정합니다
func Set(key string, value interface{}) error {
	cm := MustGetManager()
	return cm.Set(key, value)
}

// IsSet은 특정 키가 설정되었는지 확인합니다
func IsSet(key string) bool {
	cm := MustGetManager()
	return cm.IsSet(key)
}

// ResetInstance는 싱글톤 인스턴스를 리셋합니다 (테스트용)
func ResetInstance() {
	once = sync.Once{}
	instance = nil
	initErr = nil
}