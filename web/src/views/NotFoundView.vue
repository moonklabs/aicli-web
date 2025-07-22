<template>
  <div class="not-found-view">
    <div class="not-found-container">
      <!-- 404 일러스트레이션 -->
      <div class="illustration">
        <div class="error-code">404</div>
        <NIcon size="120" color="#e2e8f0">
          <SearchIcon />
        </NIcon>
      </div>

      <!-- 메시지 -->
      <div class="message-section">
        <h1 class="error-title">페이지를 찾을 수 없습니다</h1>
        <p class="error-description">
          요청하신 페이지가 존재하지 않거나 이동되었을 수 있습니다.<br>
          URL을 다시 확인해주시거나 아래 버튼을 통해 홈으로 돌아가세요.
        </p>
      </div>

      <!-- 액션 버튼들 -->
      <div class="action-buttons">
        <NButton
          type="primary"
          size="large"
          @click="goHome"
        >
          <template #icon>
            <NIcon>
              <HomeIcon />
            </NIcon>
          </template>
          홈으로 돌아가기
        </NButton>

        <NButton
          size="large"
          @click="goBack"
        >
          <template #icon>
            <NIcon>
              <ArrowBackIcon />
            </NIcon>
          </template>
          이전 페이지로
        </NButton>
      </div>

      <!-- 도움말 링크 -->
      <div class="help-links">
        <p class="help-text">다음 링크들이 도움이 될 수 있습니다:</p>
        <div class="link-grid">
          <router-link to="/" class="help-link">
            <NIcon>
              <DashboardIcon />
            </NIcon>
            대시보드
          </router-link>

          <router-link to="/workspaces" class="help-link">
            <NIcon>
              <ServerIcon />
            </NIcon>
            워크스페이스
          </router-link>

          <router-link to="/docker" class="help-link">
            <NIcon>
              <ContainerIcon />
            </NIcon>
            Docker 관리
          </router-link>

          <a href="#" class="help-link" @click="showSupport">
            <NIcon>
              <HelpCircleIcon />
            </NIcon>
            도움말
          </a>
        </div>
      </div>

      <!-- 검색 제안 -->
      <div class="search-section">
        <NDivider>또는 검색해보세요</NDivider>
        <NInputGroup>
          <NInput
            v-model:value="searchQuery"
            placeholder="찾고 계신 내용을 검색해보세요..."
            size="large"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <NIcon>
                <SearchIcon />
              </NIcon>
            </template>
          </NInput>
          <NButton
            type="primary"
            size="large"
            @click="handleSearch"
            :disabled="!searchQuery.trim()"
          >
            검색
          </NButton>
        </NInputGroup>
      </div>
    </div>

    <!-- 배경 패턴 -->
    <div class="background-pattern">
      <div class="pattern-dot" v-for="i in 50" :key="i"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NDivider,
  NIcon,
  NInput,
  NInputGroup,
  useMessage,
} from 'naive-ui'
import {
  ArrowBackOutline as ArrowBackIcon,
  CubeOutline as ContainerIcon,
  SpeedometerOutline as DashboardIcon,
  HelpCircleOutline as HelpCircleIcon,
  HomeOutline as HomeIcon,
  SearchOutline as SearchIcon,
  ServerOutline as ServerIcon,
} from '@vicons/ionicons5'

const router = useRouter()
const message = useMessage()

// 상태
const searchQuery = ref('')

// 메서드
const goHome = () => {
  router.push('/')
}

const goBack = () => {
  if (window.history.length > 1) {
    router.go(-1)
  } else {
    router.push('/')
  }
}

const handleSearch = () => {
  if (!searchQuery.value.trim()) return

  // 실제로는 검색 페이지로 이동하거나 검색 결과를 보여줌
  message.info(`"${searchQuery.value}" 검색 기능은 준비 중입니다`)
}

const showSupport = (event: Event) => {
  event.preventDefault()
  message.info('지원 센터 기능은 준비 중입니다')
}
</script>

<style lang="scss" scoped>
.not-found-view {
  position: relative;
  min-height: 100vh;
  @include flex-center;
  background: $light-bg-secondary;
  padding: $spacing-4;
  overflow: hidden;

  .dark & {
    background: $dark-bg-primary;
  }
}

.not-found-container {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 600px;
  text-align: center;
  background: $light-bg-primary;
  border-radius: $border-radius-xl;
  padding: $spacing-8;
  box-shadow: $shadow-lg;

  .dark & {
    background: $dark-bg-secondary;
  }

  @include mobile {
    padding: $spacing-6;
    margin: $spacing-4;
  }
}

.illustration {
  position: relative;
  margin-bottom: $spacing-8;

  .error-code {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    font-size: 4rem;
    font-weight: $font-weight-bold;
    color: map-get($gray-colors, 300);
    z-index: -1;

    .dark & {
      color: $dark-bg-tertiary;
    }

    @include mobile {
      font-size: 3rem;
    }
  }
}

.message-section {
  margin-bottom: $spacing-8;

  .error-title {
    font-size: $font-size-3xl;
    font-weight: $font-weight-bold;
    color: $light-text-primary;
    margin: 0 0 $spacing-4 0;

    .dark & {
      color: $dark-text-primary;
    }

    @include mobile {
      font-size: $font-size-2xl;
    }
  }

  .error-description {
    font-size: $font-size-base;
    line-height: $line-height-relaxed;
    color: $light-text-secondary;
    margin: 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.action-buttons {
  display: flex;
  gap: $spacing-4;
  justify-content: center;
  margin-bottom: $spacing-8;

  @include mobile {
    flex-direction: column;
    align-items: stretch;
  }
}

.help-links {
  margin-bottom: $spacing-8;

  .help-text {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    margin: 0 0 $spacing-4 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }

  .link-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: $spacing-3;

    @include mobile {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  .help-link {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: $spacing-2;
    padding: $spacing-3;
    border-radius: $border-radius-md;
    text-decoration: none;
    color: $light-text-secondary;
    transition: $transition-base;
    font-size: $font-size-sm;

    .dark & {
      color: $dark-text-secondary;
    }

    &:hover {
      background: map-get($gray-colors, 50);
      color: map-get($primary-colors, 600);
      transform: translateY(-2px);

      .dark & {
        background: $dark-bg-tertiary;
        color: map-get($primary-colors, 400);
      }
    }
  }
}

.search-section {
  .n-input-group {
    max-width: 400px;
    margin: 0 auto;
  }
}

.background-pattern {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  opacity: 0.1;
  pointer-events: none;

  .pattern-dot {
    position: absolute;
    width: 4px;
    height: 4px;
    background: map-get($gray-colors, 400);
    border-radius: 50%;
    animation: float 6s ease-in-out infinite;

    .dark & {
      background: $dark-text-tertiary;
    }

    @for $i from 1 through 50 {
      &:nth-child(#{$i}) {
        left: random(100) * 1%;
        top: random(100) * 1%;
        animation-delay: random(60) * 0.1s;
        animation-duration: (4 + random(40) * 0.1) * 1s;
      }
    }
  }
}

@keyframes float {
  0%, 100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-20px);
  }
}

// Naive UI 컴포넌트 스타일 오버라이드
:deep(.n-divider) {
  margin: $spacing-6 0 $spacing-4 0;

  .n-divider__title {
    color: $light-text-secondary;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}
</style>