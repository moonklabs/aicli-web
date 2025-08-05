<template>
  <div class="user-menu">
    <NDropdown
      :options="menuOptions"
      placement="bottom-end"
      trigger="click"
      @select="handleMenuSelect"
    >
      <div class="user-trigger">
        <NAvatar
          :size="32"
          :src="user.avatar"
          :fallback-src="'/default-avatar.png'"
          round
        >
          {{ userInitials }}
        </NAvatar>
        <span v-if="!isMobile" class="user-name">
          {{ user.displayName || user.username }}
        </span>
        <NIcon v-if="!isMobile" size="16" class="dropdown-arrow">
          <ChevronDownIcon />
        </NIcon>
      </div>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import {
  NAvatar,
  NDropdown,
  NIcon,
  type DropdownOption,
  useMessage,
} from 'naive-ui'
import {
  ChevronDownOutline as ChevronDownIcon,
  PersonOutline as ProfileIcon,
  SettingsOutline as SettingsIcon,
  ShieldOutline as SecurityIcon,
  LogOutOutline as LogoutIcon,
} from '@vicons/ionicons5'

import { useUserStore } from '@/stores/user'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import type { User } from '@/types/api'

interface Props {
  user: User
}

const props = defineProps<Props>()

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const { isMobile } = useMobileOptimization()

// 사용자 이니셜
const userInitials = computed(() => {
  const name = props.user.displayName || props.user.username
  return name
    .split(' ')
    .map(word => word.charAt(0))
    .join('')
    .toUpperCase()
    .slice(0, 2)
})

// 드롭다운 메뉴 옵션
const menuOptions = computed<DropdownOption[]>(() => [
  {
    label: '프로필 설정',
    key: 'profile',
    icon: () => h(NIcon, { component: ProfileIcon }),
  },
  {
    label: '계정 설정',
    key: 'settings',
    icon: () => h(NIcon, { component: SettingsIcon }),
  },
  {
    label: '보안 설정',
    key: 'security',
    icon: () => h(NIcon, { component: SecurityIcon }),
  },
  {
    type: 'divider',
    key: 'divider',
  },
  {
    label: '로그아웃',
    key: 'logout',
    icon: () => h(NIcon, { component: LogoutIcon }),
    props: {
      style: 'color: #e53e3e',
    },
  },
])

// 메뉴 선택 핸들러
const handleMenuSelect = async (key: string) => {
  switch (key) {
    case 'profile':
      router.push('/profile')
      break
    
    case 'settings':
      router.push('/profile?tab=settings')
      break
    
    case 'security':
      router.push('/security')
      break
    
    case 'logout':
      await handleLogout()
      break
  }
}

// 로그아웃 처리
const handleLogout = async () => {
  try {
    await userStore.logout()
    message.success('로그아웃되었습니다')
    router.push('/login')
  } catch (error) {
    console.error('로그아웃 실패:', error)
    message.error('로그아웃에 실패했습니다')
  }
}
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.user-menu {
  position: relative;
}

.user-trigger {
  @include flex-center;
  gap: $spacing-2;
  padding: $spacing-2;
  border-radius: $border-radius-md;
  cursor: pointer;
  transition: $transition-base;

  @include touch-target(44px);

  &:hover {
    @include no-touch {
      background: map-get($gray-colors, 100);

      .dark & {
        background: $dark-bg-tertiary;
      }
    }
  }

  .user-name {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    max-width: 120px;
    @include text-ellipsis;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .dropdown-arrow {
    color: $light-text-secondary;
    transition: transform 0.2s ease;

    .dark & {
      color: $dark-text-secondary;
    }
  }

  &:hover .dropdown-arrow {
    transform: rotate(180deg);
  }
}

// 모바일 최적화
@include mobile {
  .user-trigger {
    padding: $spacing-2;
    min-width: 44px;
    min-height: 44px;
    justify-content: center;
  }
}
</style>