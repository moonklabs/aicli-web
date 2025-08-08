package performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// CustomAllocator 커스텀 메모리 할당자
type CustomAllocator struct {
	memoryLimit   int64
	currentUsage  int64
	allocations   map[uintptr]*Allocation
	freeList      *FreeBlock
	mutex         sync.RWMutex
	stats         *AllocatorStats
}

// Allocation 할당 정보
type Allocation struct {
	Address uintptr
	Size    int
	InUse   bool
}

// FreeBlock 프리 블록 리스트
type FreeBlock struct {
	Address uintptr
	Size    int
	Next    *FreeBlock
}

// AllocatorStats 할당자 통계
type AllocatorStats struct {
	TotalAllocations   uint64
	TotalDeallocations uint64
	CurrentAllocations uint64
	TotalBytes         uint64
	FragmentationRatio float64
}

// NewCustomAllocator 새 커스텀 할당자 생성
func NewCustomAllocator(memoryLimit int64) *CustomAllocator {
	return &CustomAllocator{
		memoryLimit:  memoryLimit,
		allocations:  make(map[uintptr]*Allocation),
		stats:        &AllocatorStats{},
	}
}

// Allocate 메모리 할당
func (ca *CustomAllocator) Allocate(size int) ([]byte, error) {
	// 메모리 제한 확인
	if ca.memoryLimit > 0 && atomic.LoadInt64(&ca.currentUsage)+int64(size) > ca.memoryLimit {
		return nil, fmt.Errorf("memory limit exceeded: requested %d, limit %d, current %d",
			size, ca.memoryLimit, atomic.LoadInt64(&ca.currentUsage))
	}
	
	ca.mutex.Lock()
	defer ca.mutex.Unlock()
	
	// 프리 리스트에서 적절한 블록 찾기
	block := ca.findFreeBlock(size)
	if block != nil {
		// 재사용 가능한 블록 발견
		allocation := &Allocation{
			Address: block.Address,
			Size:    size,
			InUse:   true,
		}
		ca.allocations[block.Address] = allocation
		atomic.AddUint64(&ca.stats.CurrentAllocations, 1)
		
		// 슬라이스 생성
		return ca.makeSlice(block.Address, size), nil
	}
	
	// 새로운 할당
	data := make([]byte, size)
	addr := uintptr(unsafe.Pointer(&data[0]))
	
	allocation := &Allocation{
		Address: addr,
		Size:    size,
		InUse:   true,
	}
	
	ca.allocations[addr] = allocation
	
	// 통계 업데이트
	atomic.AddInt64(&ca.currentUsage, int64(size))
	atomic.AddUint64(&ca.stats.TotalAllocations, 1)
	atomic.AddUint64(&ca.stats.CurrentAllocations, 1)
	atomic.AddUint64(&ca.stats.TotalBytes, uint64(size))
	
	return data, nil
}

// Free 메모리 해제
func (ca *CustomAllocator) Free(data []byte) {
	if len(data) == 0 {
		return
	}
	
	addr := uintptr(unsafe.Pointer(&data[0]))
	
	ca.mutex.Lock()
	defer ca.mutex.Unlock()
	
	allocation, exists := ca.allocations[addr]
	if !exists {
		return
	}
	
	// 프리 리스트에 추가
	ca.addToFreeList(allocation)
	
	// 할당 정보 업데이트
	allocation.InUse = false
	
	// 통계 업데이트
	atomic.AddInt64(&ca.currentUsage, -int64(allocation.Size))
	atomic.AddUint64(&ca.stats.TotalDeallocations, 1)
	atomic.AddUint64(&ca.stats.CurrentAllocations, ^uint64(0))
}

// findFreeBlock 프리 리스트에서 적절한 블록 찾기
func (ca *CustomAllocator) findFreeBlock(size int) *FreeBlock {
	var prev *FreeBlock
	current := ca.freeList
	
	for current != nil {
		if current.Size >= size {
			// 적절한 블록 발견
			if prev != nil {
				prev.Next = current.Next
			} else {
				ca.freeList = current.Next
			}
			return current
		}
		prev = current
		current = current.Next
	}
	
	return nil
}

// addToFreeList 프리 리스트에 블록 추가
func (ca *CustomAllocator) addToFreeList(allocation *Allocation) {
	block := &FreeBlock{
		Address: allocation.Address,
		Size:    allocation.Size,
		Next:    ca.freeList,
	}
	
	ca.freeList = block
	
	// 인접 블록 병합 시도
	ca.coalesceBlocks()
}

// coalesceBlocks 인접 프리 블록 병합
func (ca *CustomAllocator) coalesceBlocks() {
	if ca.freeList == nil {
		return
	}
	
	// 주소순으로 정렬
	ca.sortFreeList()
	
	// 인접 블록 병합
	current := ca.freeList
	for current != nil && current.Next != nil {
		next := current.Next
		
		// 인접한지 확인
		if current.Address+uintptr(current.Size) == next.Address {
			// 병합
			current.Size += next.Size
			current.Next = next.Next
		} else {
			current = current.Next
		}
	}
}

// sortFreeList 프리 리스트를 주소순으로 정렬
func (ca *CustomAllocator) sortFreeList() {
	if ca.freeList == nil || ca.freeList.Next == nil {
		return
	}
	
	// 간단한 버블 정렬 (실제로는 더 효율적인 정렬 사용)
	changed := true
	for changed {
		changed = false
		current := ca.freeList
		var prev *FreeBlock
		
		for current != nil && current.Next != nil {
			if current.Address > current.Next.Address {
				// 스왑
				next := current.Next
				current.Next = next.Next
				next.Next = current
				
				if prev != nil {
					prev.Next = next
				} else {
					ca.freeList = next
				}
				
				changed = true
				prev = next
			} else {
				prev = current
				current = current.Next
			}
		}
	}
}

// makeSlice 주소에서 슬라이스 생성 (unsafe)
func (ca *CustomAllocator) makeSlice(addr uintptr, size int) []byte {
	// 주의: 실제 프로덕션에서는 더 안전한 방법 사용
	return (*[1 << 30]byte)(unsafe.Pointer(addr))[:size:size]
}

// GetStats 통계 조회
func (ca *CustomAllocator) GetStats() *AllocatorStats {
	ca.mutex.RLock()
	defer ca.mutex.RUnlock()
	
	// 단편화 비율 계산
	totalFreeSize := 0
	freeBlockCount := 0
	current := ca.freeList
	
	for current != nil {
		totalFreeSize += current.Size
		freeBlockCount++
		current = current.Next
	}
	
	if totalFreeSize > 0 && freeBlockCount > 0 {
		avgBlockSize := totalFreeSize / freeBlockCount
		if avgBlockSize > 0 {
			ca.stats.FragmentationRatio = float64(freeBlockCount) / float64(avgBlockSize)
		}
	}
	
	return ca.stats
}

// Defragment 메모리 조각 모음
func (ca *CustomAllocator) Defragment() {
	ca.mutex.Lock()
	defer ca.mutex.Unlock()
	
	// 프리 블록 병합
	ca.coalesceBlocks()
	
	log.Debug("Memory defragmentation completed")
}

// GetUsage 현재 메모리 사용량 조회
func (ca *CustomAllocator) GetUsage() int64 {
	return atomic.LoadInt64(&ca.currentUsage)
}

// GetLimit 메모리 제한 조회
func (ca *CustomAllocator) GetLimit() int64 {
	return ca.memoryLimit
}

// SetLimit 메모리 제한 설정
func (ca *CustomAllocator) SetLimit(limit int64) {
	ca.memoryLimit = limit
}

// Reset 할당자 리셋
func (ca *CustomAllocator) Reset() {
	ca.mutex.Lock()
	defer ca.mutex.Unlock()
	
	ca.allocations = make(map[uintptr]*Allocation)
	ca.freeList = nil
	atomic.StoreInt64(&ca.currentUsage, 0)
	
	// 통계 리셋
	ca.stats = &AllocatorStats{}
	
	log.Debug("Allocator reset completed")
}