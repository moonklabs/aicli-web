package docker

import (
	"fmt"
	"path/filepath"
	"sync"
)

// ImageOptimizer manages image-specific optimization profiles
type ImageOptimizer struct {
	profiles map[string]*ImageProfile
	mutex    sync.RWMutex
}

// ImageProfile contains image-specific configuration
type ImageProfile struct {
	ImagePattern     string
	DefaultShell     string
	Environment      map[string]string
	WorkingDir       string
	PreExecCommands  []string
	PostExecCommands []string
	ResourceLimits   *PTYResourceLimits
}

// PTYResourceLimits defines resource constraints for PTY sessions
type PTYResourceLimits struct {
	CPULimit    float64
	MemoryLimit int64
	PIDs        int
}

// NewImageOptimizer creates a new image optimizer
func NewImageOptimizer() *ImageOptimizer {
	io := &ImageOptimizer{
		profiles: make(map[string]*ImageProfile),
	}

	// 기본 프로필 설정
	io.initDefaultProfiles()
	return io
}

// initDefaultProfiles initializes default image profiles
func (io *ImageOptimizer) initDefaultProfiles() {
	// Node.js 이미지 프로필
	io.profiles["node:*"] = &ImageProfile{
		ImagePattern: "node:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"NODE_ENV": "development",
			"NPM_CONFIG_LOGLEVEL": "warn",
		},
		WorkingDir: "/app",
		PreExecCommands: []string{
			"npm config set registry https://registry.npmjs.org/",
		},
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			PIDs:        1000,
		},
	}

	// Python 이미지 프로필
	io.profiles["python:*"] = &ImageProfile{
		ImagePattern: "python:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"PYTHONUNBUFFERED": "1",
			"PYTHONDONTWRITEBYTECODE": "1",
		},
		WorkingDir: "/app",
		PreExecCommands: []string{
			"pip config set global.index-url https://pypi.org/simple",
		},
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			PIDs:        500,
		},
	}

	// Go 이미지 프로필
	io.profiles["golang:*"] = &ImageProfile{
		ImagePattern: "golang:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"GO111MODULE": "on",
			"GOPROXY":     "https://proxy.golang.org,direct",
			"CGO_ENABLED": "1",
		},
		WorkingDir: "/go/src/app",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    4.0,
			MemoryLimit: 2048 * 1024 * 1024, // 2GB
			PIDs:        1000,
		},
	}

	// Java 이미지 프로필
	io.profiles["openjdk:*"] = &ImageProfile{
		ImagePattern: "openjdk:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"JAVA_OPTS": "-Xmx512m -Xms256m",
		},
		WorkingDir: "/app",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 2048 * 1024 * 1024, // 2GB
			PIDs:        500,
		},
	}

	// Ruby 이미지 프로필
	io.profiles["ruby:*"] = &ImageProfile{
		ImagePattern: "ruby:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"BUNDLE_PATH": "/usr/local/bundle",
			"GEM_HOME":    "/usr/local/bundle",
		},
		WorkingDir: "/app",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			PIDs:        500,
		},
	}

	// Alpine Linux 이미지 프로필
	io.profiles["alpine:*"] = &ImageProfile{
		ImagePattern: "alpine:*",
		DefaultShell: "/bin/sh",
		Environment: map[string]string{
			"LANG":   "C.UTF-8",
			"LC_ALL": "C.UTF-8",
		},
		WorkingDir: "/",
		PreExecCommands: []string{
			"apk update",
		},
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    1.0,
			MemoryLimit: 512 * 1024 * 1024, // 512MB
			PIDs:        200,
		},
	}

	// Ubuntu/Debian 이미지 프로필
	io.profiles["ubuntu:*"] = &ImageProfile{
		ImagePattern: "ubuntu:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"DEBIAN_FRONTEND": "noninteractive",
			"LANG":            "C.UTF-8",
			"LC_ALL":          "C.UTF-8",
		},
		WorkingDir: "/",
		PreExecCommands: []string{
			"apt-get update",
		},
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			PIDs:        500,
		},
	}

	// Nginx 이미지 프로필
	io.profiles["nginx:*"] = &ImageProfile{
		ImagePattern: "nginx:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"NGINX_HOST": "localhost",
			"NGINX_PORT": "80",
		},
		WorkingDir: "/usr/share/nginx/html",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    1.0,
			MemoryLimit: 256 * 1024 * 1024, // 256MB
			PIDs:        100,
		},
	}

	// Redis 이미지 프로필
	io.profiles["redis:*"] = &ImageProfile{
		ImagePattern: "redis:*",
		DefaultShell: "/bin/bash",
		Environment:  map[string]string{},
		WorkingDir:   "/data",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    1.0,
			MemoryLimit: 512 * 1024 * 1024, // 512MB
			PIDs:        50,
		},
	}

	// PostgreSQL 이미지 프로필
	io.profiles["postgres:*"] = &ImageProfile{
		ImagePattern: "postgres:*",
		DefaultShell: "/bin/bash",
		Environment: map[string]string{
			"PGDATA": "/var/lib/postgresql/data",
		},
		WorkingDir: "/",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			PIDs:        200,
		},
	}
}

// GetProfileForImage returns the profile for a given image
func (io *ImageOptimizer) GetProfileForImage(imageName string) *ImageProfile {
	io.mutex.RLock()
	defer io.mutex.RUnlock()

	// 정확한 매칭 먼저 확인
	if profile, exists := io.profiles[imageName]; exists {
		return profile
	}

	// 패턴 매칭
	for pattern, profile := range io.profiles {
		if matched, _ := filepath.Match(pattern, imageName); matched {
			return profile
		}
	}

	return io.getDefaultProfile()
}

// getDefaultProfile returns the default profile
func (io *ImageOptimizer) getDefaultProfile() *ImageProfile {
	return &ImageProfile{
		ImagePattern: "*",
		DefaultShell: "/bin/sh",
		Environment: map[string]string{
			"LANG":   "C.UTF-8",
			"LC_ALL": "C.UTF-8",
		},
		WorkingDir: "/",
		ResourceLimits: &PTYResourceLimits{
			CPULimit:    1.0,
			MemoryLimit: 512 * 1024 * 1024, // 512MB
			PIDs:        100,
		},
	}
}

// AddProfile adds a custom profile
func (io *ImageOptimizer) AddProfile(pattern string, profile *ImageProfile) {
	io.mutex.Lock()
	defer io.mutex.Unlock()

	profile.ImagePattern = pattern
	io.profiles[pattern] = profile
}

// RemoveProfile removes a profile
func (io *ImageOptimizer) RemoveProfile(pattern string) {
	io.mutex.Lock()
	defer io.mutex.Unlock()

	delete(io.profiles, pattern)
}

// GetAllProfiles returns all profiles
func (io *ImageOptimizer) GetAllProfiles() map[string]*ImageProfile {
	io.mutex.RLock()
	defer io.mutex.RUnlock()

	allProfiles := make(map[string]*ImageProfile)
	for pattern, profile := range io.profiles {
		allProfiles[pattern] = profile
	}
	return allProfiles
}

// UpdateProfile updates an existing profile
func (io *ImageOptimizer) UpdateProfile(pattern string, profile *ImageProfile) error {
	io.mutex.Lock()
	defer io.mutex.Unlock()

	if _, exists := io.profiles[pattern]; !exists {
		return fmt.Errorf("profile not found: %s", pattern)
	}

	profile.ImagePattern = pattern
	io.profiles[pattern] = profile
	return nil
}