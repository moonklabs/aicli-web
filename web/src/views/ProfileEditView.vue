<template>
  <div class="profile-edit">
    <!-- 헤더 섹션 -->
    <div class="header">
      <div class="title-section">
        <h1>프로파일 설정</h1>
        <p class="subtitle">개인 정보, 보안 설정 및 계정 정보를 관리합니다</p>
      </div>
      
      <div class="actions-section">
        <n-space>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!hasChanges"
            @click="saveAllChanges"
          >
            <template #icon>
              <n-icon><Save /></n-icon>
            </template>
            모든 변경사항 저장
          </n-button>
          <n-button
            type="default"
            ghost
            :disabled="!hasChanges"
            @click="discardChanges"
          >
            <template #icon>
              <n-icon><Refresh /></n-icon>
            </template>
            변경사항 취소
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- 메인 콘텐츠 -->
    <div class="content">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <!-- 기본 정보 탭 -->
        <n-tab-pane name="basic" tab="기본 정보">
          <div class="profile-basic">
            <!-- 프로파일 이미지 섹션 -->
            <div class="profile-image-section">
              <h3>프로파일 이미지</h3>
              <ProfileImageUpload
                :current-image="profile?.avatar"
                :uploading="uploadingImage"
                @upload="handleImageUpload"
                @delete="handleImageDelete"
              />
            </div>

            <!-- 기본 정보 폼 -->
            <div class="basic-info-form">
              <h3>기본 정보</h3>
              <n-form
                ref="basicFormRef"
                :model="basicForm"
                :rules="basicFormRules"
                label-placement="top"
                require-mark-placement="right-hanging"
              >
                <n-grid :cols="24" :x-gap="16" :y-gap="16">
                  <n-grid-item :span="12">
                    <n-form-item label="표시 이름" path="displayName">
                      <n-input
                        v-model:value="basicForm.displayName"
                        placeholder="표시할 이름을 입력하세요"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>
                  
                  <n-grid-item :span="6">
                    <n-form-item label="이름" path="firstName">
                      <n-input
                        v-model:value="basicForm.firstName"
                        placeholder="이름"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>
                  
                  <n-grid-item :span="6">
                    <n-form-item label="성" path="lastName">
                      <n-input
                        v-model:value="basicForm.lastName"
                        placeholder="성"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="24">
                    <n-form-item label="자기소개" path="bio">
                      <n-input
                        v-model:value="basicForm.bio"
                        type="textarea"
                        :rows="3"
                        placeholder="간단한 자기소개를 작성해주세요"
                        show-count
                        :maxlength="500"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="12">
                    <n-form-item label="이메일" path="email">
                      <n-input-group>
                        <n-input
                          :value="profile?.email"
                          readonly
                          placeholder="이메일 주소"
                        />
                        <n-button
                          type="primary"
                          ghost
                          @click="requestEmailChange"
                        >
                          변경
                        </n-button>
                      </n-input-group>
                      <template #feedback>
                        <n-space size="small" align="center">
                          <n-icon
                            :color="profile?.isEmailVerified ? '#18a058' : '#f0a020'"
                            size="14"
                          >
                            <CheckmarkCircle v-if="profile?.isEmailVerified" />
                            <AlertCircle v-else />
                          </n-icon>
                          <n-text :depth="3" style="font-size: 12px;">
                            {{ profile?.isEmailVerified ? '인증됨' : '인증 필요' }}
                          </n-text>
                          <n-button
                            v-if="!profile?.isEmailVerified"
                            text
                            size="tiny"
                            type="primary"
                            @click="requestEmailVerification"
                          >
                            인증 메일 발송
                          </n-button>
                        </n-space>
                      </template>
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="12">
                    <n-form-item label="전화번호" path="phone">
                      <n-input-group>
                        <n-input
                          v-model:value="basicForm.phone"
                          placeholder="전화번호"
                          @blur="autoSave('basic')"
                        />
                        <n-button
                          v-if="basicForm.phone && !profile?.isPhoneVerified"
                          type="primary"
                          ghost
                          @click="requestPhoneVerification"
                        >
                          인증
                        </n-button>
                      </n-input-group>
                      <template #feedback>
                        <n-space v-if="basicForm.phone" size="small" align="center">
                          <n-icon
                            :color="profile?.isPhoneVerified ? '#18a058' : '#f0a020'"
                            size="14"
                          >
                            <CheckmarkCircle v-if="profile?.isPhoneVerified" />
                            <AlertCircle v-else />
                          </n-icon>
                          <n-text :depth="3" style="font-size: 12px;">
                            {{ profile?.isPhoneVerified ? '인증됨' : '인증 필요' }}
                          </n-text>
                        </n-space>
                      </template>
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="8">
                    <n-form-item label="생년월일" path="birthDate">
                      <n-date-picker
                        v-model:value="birthDateValue"
                        type="date"
                        placeholder="생년월일 선택"
                        style="width: 100%"
                        @update:value="handleBirthDateChange"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="8">
                    <n-form-item label="시간대" path="timezone">
                      <n-select
                        v-model:value="basicForm.timezone"
                        :options="timezoneOptions"
                        filterable
                        placeholder="시간대 선택"
                        @update:value="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="8">
                    <n-form-item label="언어" path="language">
                      <n-select
                        v-model:value="basicForm.language"
                        :options="languageOptions"
                        placeholder="언어 선택"
                        @update:value="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="12">
                    <n-form-item label="웹사이트" path="website">
                      <n-input
                        v-model:value="basicForm.website"
                        placeholder="https://example.com"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="12">
                    <n-form-item label="위치" path="location">
                      <n-input
                        v-model:value="basicForm.location"
                        placeholder="도시, 국가"
                        @blur="autoSave('basic')"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="24">
                    <n-form-item label="테마" path="theme">
                      <n-radio-group v-model:value="basicForm.theme" @update:value="autoSave('basic')">
                        <n-space>
                          <n-radio value="light">라이트</n-radio>
                          <n-radio value="dark">다크</n-radio>
                          <n-radio value="auto">시스템 설정</n-radio>
                        </n-space>
                      </n-radio-group>
                    </n-form-item>
                  </n-grid-item>
                </n-grid>
              </n-form>
            </div>
          </div>
        </n-tab-pane>

        <!-- 보안 설정 탭 -->
        <n-tab-pane name="security" tab="보안 설정">
          <SecuritySettingsPanel
            :loading="loadingSecurity"
            @password-change="handlePasswordChange"
            @two-factor-setup="handleTwoFactorSetup"
          />
        </n-tab-pane>

        <!-- 알림 설정 탭 -->
        <n-tab-pane name="notifications" tab="알림 설정">
          <NotificationSettings
            :settings="notificationSettings"
            :loading="loadingNotifications"
            @update="handleNotificationUpdate"
          />
        </n-tab-pane>

        <!-- 개인정보 설정 탭 -->
        <n-tab-pane name="privacy" tab="개인정보">
          <PrivacySettingsPanel
            :settings="privacySettings"
            :loading="loadingPrivacy"
            @update="handlePrivacyUpdate"
          />
        </n-tab-pane>

        <!-- 계정 관리 탭 -->
        <n-tab-pane name="account" tab="계정 관리">
          <AccountDangerZone
            :profile="profile"
            @deactivate="handleAccountDeactivation"
            @delete="handleAccountDeletion"
          />
        </n-tab-pane>
      </n-tabs>
    </div>

    <!-- 이메일 변경 모달 -->
    <EmailChangeModal
      v-model:show="showEmailChangeModal"
      @confirm="handleEmailChangeConfirm"
    />

    <!-- 전화번호 인증 모달 -->
    <PhoneVerificationModal
      v-model:show="showPhoneVerificationModal"
      :phone="basicForm.phone"
      @verify="handlePhoneVerificationConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { useMessage, useDialog, useLoadingBar } from 'naive-ui'
import {
  SaveSharp as Save,
  RefreshSharp as Refresh,
  CheckmarkCircleSharp as CheckmarkCircle,
  AlertCircleSharp as AlertCircle
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import { useUserStore } from '@/stores/user'
import type { 
  UserProfile, 
  UpdateProfileRequest,
  NotificationSettings,
  PrivacySettings
} from '@/types/api'

// 컴포넌트 import (아직 구현되지 않음)
import ProfileImageUpload from '@/components/Profile/ProfileImageUpload.vue'
import SecuritySettingsPanel from '@/components/Profile/SecuritySettingsPanel.vue'
import NotificationSettings from '@/components/Profile/NotificationSettings.vue'
import PrivacySettingsPanel from '@/components/Profile/PrivacySettingsPanel.vue'
import AccountDangerZone from '@/components/Profile/AccountDangerZone.vue'
import EmailChangeModal from '@/components/Profile/EmailChangeModal.vue'
import PhoneVerificationModal from '@/components/Profile/PhoneVerificationModal.vue'

// 컴포저블
const message = useMessage()
const dialog = useDialog()
const loadingBar = useLoadingBar()
const userStore = useUserStore()

// 반응형 상태
const activeTab = ref('basic')
const loading = ref(false)
const saving = ref(false)
const uploadingImage = ref(false)
const loadingSecurity = ref(false)
const loadingNotifications = ref(false)
const loadingPrivacy = ref(false)

// 데이터 상태
const profile = ref<UserProfile | null>(null)
const notificationSettings = ref<NotificationSettings | null>(null)
const privacySettings = ref<PrivacySettings | null>(null)
const originalProfile = ref<UserProfile | null>(null)

// 모달 상태
const showEmailChangeModal = ref(false)
const showPhoneVerificationModal = ref(false)

// 폼 참조
const basicFormRef = ref()

// 기본 정보 폼 데이터
const basicForm = reactive<UpdateProfileRequest>({
  displayName: '',
  firstName: '',
  lastName: '',
  bio: '',
  phone: '',
  birthDate: '',
  website: '',
  location: '',
  timezone: '',
  language: '',
  theme: 'auto'
})

// 생년월일 값 (날짜 피커용)
const birthDateValue = ref<number | null>(null)

// 폼 검증 규칙
const basicFormRules = {
  displayName: {
    required: true,
    message: '표시 이름을 입력해주세요',
    trigger: ['blur', 'input']
  },
  website: {
    pattern: /^https?:\/\/.+/,
    message: '올바른 URL 형식을 입력해주세요 (http:// 또는 https://)',
    trigger: ['blur']
  }
}

// 옵션 데이터
const timezoneOptions = [
  { label: 'Asia/Seoul (GMT+9)', value: 'Asia/Seoul' },
  { label: 'America/New_York (GMT-5)', value: 'America/New_York' },
  { label: 'Europe/London (GMT+0)', value: 'Europe/London' },
  { label: 'Asia/Tokyo (GMT+9)', value: 'Asia/Tokyo' },
  { label: 'Australia/Sydney (GMT+11)', value: 'Australia/Sydney' }
]

const languageOptions = [
  { label: '한국어', value: 'ko' },
  { label: 'English', value: 'en' },
  { label: '日本語', value: 'ja' },
  { label: '中文', value: 'zh' }
]

// 계산된 속성
const hasChanges = computed(() => {
  if (!originalProfile.value) return false
  
  const original = originalProfile.value
  return (
    basicForm.displayName !== original.displayName ||
    basicForm.firstName !== original.firstName ||
    basicForm.lastName !== original.lastName ||
    basicForm.bio !== original.bio ||
    basicForm.phone !== original.phone ||
    basicForm.birthDate !== original.birthDate ||
    basicForm.website !== original.website ||
    basicForm.location !== original.location ||
    basicForm.timezone !== original.timezone ||
    basicForm.language !== original.language ||
    basicForm.theme !== original.theme
  )
})

// 메서드
const loadProfileData = async () => {
  loading.value = true
  try {
    const [profileData, notificationData, privacyData] = await Promise.all([
      profileApi.getProfile(),
      profileApi.getNotificationSettings(),
      profileApi.getPrivacySettings()
    ])
    
    profile.value = profileData
    originalProfile.value = { ...profileData }
    notificationSettings.value = notificationData
    privacySettings.value = privacyData
    
    // 폼에 데이터 설정
    Object.assign(basicForm, {
      displayName: profileData.displayName || '',
      firstName: profileData.firstName || '',
      lastName: profileData.lastName || '',
      bio: profileData.bio || '',
      phone: profileData.phone || '',
      birthDate: profileData.birthDate || '',
      website: profileData.website || '',
      location: profileData.location || '',
      timezone: profileData.timezone || 'Asia/Seoul',
      language: profileData.language || 'ko',
      theme: profileData.theme || 'auto'
    })
    
    // 생년월일 설정
    if (profileData.birthDate) {
      birthDateValue.value = new Date(profileData.birthDate).getTime()
    }
    
  } catch (error) {
    console.error('프로파일 데이터 로드 실패:', error)
    message.error('프로파일 정보를 불러오는데 실패했습니다')
  } finally {
    loading.value = false
  }
}

const autoSave = async (section: string) => {
  if (!hasChanges.value) return
  
  try {
    // 기본 정보 자동 저장
    if (section === 'basic') {
      await basicFormRef.value?.validate()
      const updatedProfile = await profileApi.updateProfile(basicForm)
      profile.value = updatedProfile
      originalProfile.value = { ...updatedProfile }
      
      // 사용자 스토어 업데이트
      userStore.updateUser({
        displayName: updatedProfile.displayName,
        avatar: updatedProfile.avatar
      })
      
      message.success('변경사항이 자동으로 저장되었습니다', { duration: 2000 })
    }
  } catch (error) {
    console.error('자동 저장 실패:', error)
    message.error('자동 저장에 실패했습니다')
  }
}

const saveAllChanges = async () => {
  saving.value = true
  loadingBar.start()
  
  try {
    await basicFormRef.value?.validate()
    
    const updatedProfile = await profileApi.updateProfile(basicForm)
    profile.value = updatedProfile
    originalProfile.value = { ...updatedProfile }
    
    // 사용자 스토어 업데이트
    userStore.updateUser({
      displayName: updatedProfile.displayName,
      avatar: updatedProfile.avatar
    })
    
    message.success('모든 변경사항이 저장되었습니다')
    loadingBar.finish()
  } catch (error) {
    console.error('저장 실패:', error)
    message.error('저장에 실패했습니다')
    loadingBar.error()
  } finally {
    saving.value = false
  }
}

const discardChanges = () => {
  dialog.warning({
    title: '변경사항 취소',
    content: '저장하지 않은 모든 변경사항이 취소됩니다. 계속하시겠습니까?',
    positiveText: '취소',
    negativeText: '돌아가기',
    onPositiveClick: () => {
      if (originalProfile.value) {
        Object.assign(basicForm, {
          displayName: originalProfile.value.displayName || '',
          firstName: originalProfile.value.firstName || '',
          lastName: originalProfile.value.lastName || '',
          bio: originalProfile.value.bio || '',
          phone: originalProfile.value.phone || '',
          birthDate: originalProfile.value.birthDate || '',
          website: originalProfile.value.website || '',
          location: originalProfile.value.location || '',
          timezone: originalProfile.value.timezone || 'Asia/Seoul',
          language: originalProfile.value.language || 'ko',
          theme: originalProfile.value.theme || 'auto'
        })
        
        if (originalProfile.value.birthDate) {
          birthDateValue.value = new Date(originalProfile.value.birthDate).getTime()
        } else {
          birthDateValue.value = null
        }
      }
      message.info('변경사항이 취소되었습니다')
    }
  })
}

const handleBirthDateChange = (value: number | null) => {
  if (value) {
    basicForm.birthDate = new Date(value).toISOString().split('T')[0]
  } else {
    basicForm.birthDate = ''
  }
  nextTick(() => autoSave('basic'))
}

const handleImageUpload = async (file: File, cropData?: any) => {
  uploadingImage.value = true
  try {
    const response = await profileApi.uploadProfileImage(file, cropData)
    
    if (profile.value) {
      profile.value.avatar = response.imageUrl
    }
    
    // 사용자 스토어 업데이트
    userStore.updateUser({
      avatar: response.imageUrl
    })
    
    message.success('프로파일 이미지가 업데이트되었습니다')
  } catch (error) {
    console.error('이미지 업로드 실패:', error)
    message.error('이미지 업로드에 실패했습니다')
  } finally {
    uploadingImage.value = false
  }
}

const handleImageDelete = async () => {
  try {
    await profileApi.deleteProfileImage()
    
    if (profile.value) {
      profile.value.avatar = undefined
    }
    
    // 사용자 스토어 업데이트
    userStore.updateUser({
      avatar: undefined
    })
    
    message.success('프로파일 이미지가 삭제되었습니다')
  } catch (error) {
    console.error('이미지 삭제 실패:', error)
    message.error('이미지 삭제에 실패했습니다')
  }
}

const requestEmailChange = () => {
  showEmailChangeModal.value = true
}

const handleEmailChangeConfirm = async (newEmail: string, password: string) => {
  try {
    await profileApi.requestEmailChange(newEmail, password)
    message.success('이메일 변경 확인 메일을 발송했습니다')
    showEmailChangeModal.value = false
  } catch (error) {
    console.error('이메일 변경 요청 실패:', error)
    message.error('이메일 변경 요청에 실패했습니다')
  }
}

const requestEmailVerification = async () => {
  try {
    await profileApi.requestEmailVerification()
    message.success('인증 메일을 발송했습니다')
  } catch (error) {
    console.error('이메일 인증 요청 실패:', error)
    message.error('인증 메일 발송에 실패했습니다')
  }
}

const requestPhoneVerification = () => {
  if (!basicForm.phone) {
    message.warning('먼저 전화번호를 입력해주세요')
    return
  }
  showPhoneVerificationModal.value = true
}

const handlePhoneVerificationConfirm = async (code: string) => {
  try {
    await profileApi.confirmPhoneVerification(basicForm.phone!, code)
    if (profile.value) {
      profile.value.isPhoneVerified = true
    }
    message.success('전화번호가 인증되었습니다')
    showPhoneVerificationModal.value = false
  } catch (error) {
    console.error('전화번호 인증 실패:', error)
    message.error('전화번호 인증에 실패했습니다')
  }
}

const handlePasswordChange = (data: any) => {
  // SecuritySettingsPanel에서 처리
}

const handleTwoFactorSetup = (data: any) => {
  // SecuritySettingsPanel에서 처리
}

const handleNotificationUpdate = async (settings: any) => {
  try {
    notificationSettings.value = await profileApi.updateNotificationSettings(settings)
    message.success('알림 설정이 업데이트되었습니다')
  } catch (error) {
    console.error('알림 설정 업데이트 실패:', error)
    message.error('알림 설정 업데이트에 실패했습니다')
  }
}

const handlePrivacyUpdate = async (settings: any) => {
  try {
    privacySettings.value = await profileApi.updatePrivacySettings(settings)
    message.success('개인정보 설정이 업데이트되었습니다')
  } catch (error) {
    console.error('개인정보 설정 업데이트 실패:', error)
    message.error('개인정보 설정 업데이트에 실패했습니다')
  }
}

const handleAccountDeactivation = async (data: any) => {
  // AccountDangerZone에서 처리
}

const handleAccountDeletion = async (data: any) => {
  // AccountDangerZone에서 처리
}

// 라이프사이클
onMounted(() => {
  loadProfileData()
})

// 탭 변경 시 자동 저장
watch(activeTab, (newTab, oldTab) => {
  if (oldTab === 'basic' && hasChanges.value) {
    autoSave('basic')
  }
})
</script>

<style scoped lang="scss">
.profile-edit {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;

  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 32px;
    padding-bottom: 24px;
    border-bottom: 1px solid var(--border-color);

    .title-section {
      h1 {
        margin: 0 0 8px 0;
        font-size: 28px;
        font-weight: 600;
        color: var(--text-color-1);
      }

      .subtitle {
        margin: 0;
        color: var(--text-color-2);
        font-size: 16px;
      }
    }

    .actions-section {
      flex-shrink: 0;
    }
  }

  .content {
    .profile-basic {
      .profile-image-section {
        margin-bottom: 32px;
        padding-bottom: 24px;
        border-bottom: 1px solid var(--border-color);

        h3 {
          margin: 0 0 16px 0;
          font-size: 18px;
          font-weight: 500;
          color: var(--text-color-1);
        }
      }

      .basic-info-form {
        h3 {
          margin: 0 0 16px 0;
          font-size: 18px;
          font-weight: 500;
          color: var(--text-color-1);
        }
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .profile-edit {
    padding: 16px;

    .header {
      flex-direction: column;
      gap: 16px;
      align-items: stretch;

      .actions-section {
        .n-space {
          :deep(.n-space-item) {
            flex: 1;
            
            .n-button {
              width: 100%;
            }
          }
        }
      }
    }

    .content {
      .profile-basic {
        .basic-info-form {
          :deep(.n-grid) {
            .n-grid-item {
              &[data-span="12"],
              &[data-span="8"],
              &[data-span="6"] {
                span: 24;
              }
            }
          }
        }
      }
    }
  }
}
</style>