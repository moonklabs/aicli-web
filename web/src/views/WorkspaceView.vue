<template>
  <MainLayout>
    <template #header-actions>
      <!-- 모바일 헤더 액션 버튼들 -->
      <button
        class="header-action-btn"
        @click="openCreateModal"
        aria-label="새 워크스페이스 생성"
      >
        <Icon name="Plus" />
      </button>
      <button
        class="header-action-btn"
        @click="refreshWorkspaces"
        aria-label="워크스페이스 새로고침"
      >
        <Icon name="RefreshCw" />
      </button>
    </template>

    <!-- 워크스페이스 콘텐츠 -->
    <div class="workspace-view">
      <div class="workspace-view__header">
        <h1 class="workspace-view__title">워크스페이스</h1>
        <p class="workspace-view__description">
          프로젝트별로 독립된 AI 작업 환경을 관리하세요
        </p>
      </div>

      <WorkspaceList
        @create-workspace="openCreateModal"
        @refresh="refreshWorkspaces"
      />
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import WorkspaceList from '@/components/Workspace/WorkspaceList.vue'
import Icon from '@/components/common/Icon.vue'

// 워크스페이스 생성 모달 상태
const showCreateModal = ref(false)

// 새 워크스페이스 생성 모달 열기
const openCreateModal = () => {
  showCreateModal.value = true
}

// 워크스페이스 목록 새로고침
const refreshWorkspaces = () => {
  // WorkspaceList 컴포넌트의 새로고침 메서드 호출
  console.log('워크스페이스 새로고침')
}
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.workspace-view {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;

  &__header {
    margin-bottom: $spacing-8;
    text-align: center;

    @include mobile {
      margin-bottom: $spacing-6;
      text-align: left;
    }
  }

  &__title {
    @include responsive-text($font-size-2xl, $font-size-4xl);
    font-weight: $font-weight-bold;
    color: $light-text-primary;
    margin: 0 0 $spacing-4;

    .dark & {
      color: $dark-text-primary;
    }

    @include mobile {
      margin-bottom: $spacing-2;
    }
  }

  &__description {
    @include responsive-text($font-size-base, $font-size-lg);
    color: $light-text-secondary;
    margin: 0;
    max-width: 600px;
    margin: 0 auto;

    .dark & {
      color: $dark-text-secondary;
    }

    @include mobile {
      margin: 0;
    }
  }
}

// 헤더 액션 버튼 스타일
.header-action-btn {
  @include button-base;
  @include touch-target(44px);
  padding: $spacing-2;
  background: transparent;
  color: $light-text-primary;
  border: 1px solid map-get($gray-colors, 300);
  border-radius: $border-radius-md;

  .dark & {
    color: $dark-text-primary;
    border-color: $dark-bg-tertiary;
  }

  &:hover {
    background: map-get($gray-colors, 100);
    border-color: map-get($primary-colors, 400);

    .dark & {
      background: $dark-bg-tertiary;
    }
  }

  &:active {
    transform: scale(0.95);
  }

  // 모바일에서 아이콘 크기 조정
  @include mobile {
    padding: $spacing-3;

    svg {
      width: 20px;
      height: 20px;
    }
  }
}

// 반응형 조정
@include mobile {
  .workspace-view {
    padding: 0; // MainLayout에서 이미 패딩 적용
  }
}

@include tablet {
  .workspace-view {
    &__header {
      text-align: center;
    }
  }
}
</style>