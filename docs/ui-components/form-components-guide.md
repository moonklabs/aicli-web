# 폼 컴포넌트 사용 가이드

AICLI-Web 프로젝트의 고급 UI 폼 컴포넌트 사용법을 안내합니다.

## 목차

- [개요](#개요)
- [AppButton](#appbutton)
- [AppInput](#appinput)
- [AppTextarea](#apptextarea)
- [AppSelect](#appselect)
- [AppFormField](#appformfield)
- [접근성 가이드](#접근성-가이드)
- [테마 및 커스터마이징](#테마-및-커스터마이징)

## 개요

### 주요 특징

- **완전한 접근성 지원**: WCAG 2.1 AA 수준 준수
- **TypeScript 지원**: 완전한 타입 안전성
- **다크 모드 지원**: 자동 테마 전환
- **반응형 디자인**: 모든 디바이스에서 최적화
- **커스터마이징 가능**: CSS Variables를 통한 쉬운 테마 적용
- **프레임워크 독립적**: Vue 3 Composition API 기반

### 설치 및 설정

```typescript
// 개별 컴포넌트 import
import { AppButton, AppInput, AppFormField } from '@/components/ui/form';

// 전체 import
import * as FormComponents from '@/components/ui/form';
```

## AppButton

### 기본 사용법

```vue
<template>
  <div>
    <!-- 기본 버튼 -->
    <AppButton>기본 버튼</AppButton>
    
    <!-- Primary 버튼 -->
    <AppButton type="primary">확인</AppButton>
    
    <!-- 로딩 상태 -->
    <AppButton :loading="isLoading" @click="handleSubmit">
      저장하기
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AppButton } from '@/components/ui/form';

const isLoading = ref(false);

const handleSubmit = async () => {
  isLoading.value = true;
  try {
    // API 호출 등
    await submitData();
  } finally {
    isLoading.value = false;
  }
};
</script>
```

### Props

| 속성 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `type` | `'default' \| 'primary' \| 'success' \| 'warning' \| 'error' \| 'info'` | `'default'` | 버튼의 의미적 타입 |
| `size` | `'small' \| 'medium' \| 'large'` | `'medium'` | 버튼 크기 |
| `variant` | `'solid' \| 'outline' \| 'ghost' \| 'text'` | `'solid'` | 버튼 스타일 변형 |
| `disabled` | `boolean` | `false` | 비활성화 상태 |
| `loading` | `boolean` | `false` | 로딩 상태 |
| `block` | `boolean` | `false` | 전체 너비 차지 |
| `round` | `boolean` | `false` | 둥근 모서리 |
| `circle` | `boolean` | `false` | 원형 버튼 |
| `htmlType` | `'button' \| 'submit' \| 'reset'` | `'button'` | HTML 버튼 타입 |

### 이벤트

| 이벤트 | 타입 | 설명 |
|--------|------|------|
| `click` | `(event: Event) => void` | 클릭 이벤트 |
| `focus` | `(event: FocusEvent) => void` | 포커스 이벤트 |
| `blur` | `(event: FocusEvent) => void` | 블러 이벤트 |

### 슬롯

```vue
<template>
  <!-- 아이콘과 함께 사용 -->
  <AppButton type="primary">
    <template #icon>
      <SaveIcon />
    </template>
    저장하기
  </AppButton>
  
  <!-- 접미사 아이콘 -->
  <AppButton variant="outline">
    다운로드
    <template #suffix>
      <DownloadIcon />
    </template>
  </AppButton>
  
  <!-- 아이콘만 있는 버튼 -->
  <AppButton :circle="true" aria-label="설정">
    <template #icon>
      <SettingsIcon />
    </template>
  </AppButton>
</template>
```

### 접근성

```vue
<template>
  <!-- ARIA 속성 활용 -->
  <AppButton
    type="error"
    aria-label="사용자 계정 삭제"
    aria-describedby="delete-warning"
    @click="handleDelete"
  >
    <template #icon>
      <TrashIcon />
    </template>
    삭제
  </AppButton>
  
  <p id="delete-warning" class="sr-only">
    이 작업은 되돌릴 수 없습니다.
  </p>
</template>
```

## AppInput

### 기본 사용법

```vue
<template>
  <div>
    <!-- v-model 사용 -->
    <AppInput 
      v-model:value="username"
      placeholder="사용자명을 입력하세요"
    />
    
    <!-- 이벤트 핸들링 -->
    <AppInput
      :model-value="email"
      type="email"
      placeholder="이메일 주소"
      @update:value="handleEmailChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AppInput } from '@/components/ui/form';

const username = ref('');
const email = ref('');

const handleEmailChange = (value: string) => {
  email.value = value;
  // 추가 로직...
};
</script>
```

### Props

| 속성 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `modelValue` | `string \| number` | `undefined` | 입력 값 |
| `type` | `'text' \| 'password' \| 'email' \| 'number' \| 'tel' \| 'url'` | `'text'` | 입력 필드 타입 |
| `size` | `'small' \| 'medium' \| 'large'` | `'medium'` | 입력 필드 크기 |
| `status` | `'default' \| 'success' \| 'warning' \| 'error'` | `'default'` | 상태 표시 |
| `placeholder` | `string` | `undefined` | 플레이스홀더 텍스트 |
| `disabled` | `boolean` | `false` | 비활성화 상태 |
| `readonly` | `boolean` | `false` | 읽기 전용 상태 |
| `clearable` | `boolean` | `false` | 클리어 버튼 표시 |
| `showCount` | `boolean` | `false` | 문자 수 표시 |
| `maxlength` | `number` | `undefined` | 최대 문자 수 |
| `round` | `boolean` | `false` | 둥근 모서리 |

### 고급 기능

```vue
<template>
  <!-- 클리어 기능 -->
  <AppInput
    v-model:value="searchQuery"
    :clearable="true"
    placeholder="검색어를 입력하세요"
    @clear="handleClear"
  />
  
  <!-- 문자 수 제한 -->
  <AppInput
    v-model:value="description"
    :show-count="true"
    :maxlength="100"
    placeholder="설명을 입력하세요 (최대 100자)"
  />
  
  <!-- 비밀번호 표시/숨기기 -->
  <AppInput
    v-model:value="password"
    type="password"
    placeholder="비밀번호"
  />
  
  <!-- Prefix/Suffix 슬롯 -->
  <AppInput v-model:value="amount" type="number">
    <template #prefix>
      <span class="text-gray-500">₩</span>
    </template>
    <template #suffix>
      <span class="text-gray-500">원</span>
    </template>
  </AppInput>
</template>
```

## AppTextarea

### 기본 사용법

```vue
<template>
  <AppTextarea
    v-model:value="content"
    placeholder="내용을 입력하세요"
    :rows="5"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AppTextarea } from '@/components/ui/form';

const content = ref('');
</script>
```

### 자동 크기 조정

```vue
<template>
  <!-- 자동 크기 조정 -->
  <AppTextarea
    v-model:value="autoContent"
    :autosize="true"
    :min-rows="3"
    :max-rows="10"
    placeholder="내용이 늘어나면 자동으로 크기가 조정됩니다"
  />
  
  <!-- 크기 조정 불가 -->
  <AppTextarea
    v-model:value="fixedContent"
    :resizable="false"
    :rows="4"
    placeholder="크기 조정이 불가능한 텍스트 영역"
  />
</template>
```

## AppSelect

### 기본 사용법

```vue
<template>
  <div>
    <!-- 단일 선택 -->
    <AppSelect
      v-model:value="selectedCountry"
      :options="countries"
      placeholder="국가를 선택하세요"
    />
    
    <!-- 다중 선택 -->
    <AppSelect
      v-model:value="selectedTags"
      :options="tags"
      :multiple="true"
      placeholder="태그를 선택하세요"
    />
    
    <!-- 검색 가능 -->
    <AppSelect
      v-model:value="selectedUser"
      :options="users"
      :searchable="true"
      placeholder="사용자를 검색하세요"
      @search="handleUserSearch"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AppSelect } from '@/components/ui/form';

const selectedCountry = ref('');
const selectedTags = ref([]);
const selectedUser = ref('');

const countries = [
  { label: '대한민국', value: 'kr' },
  { label: '미국', value: 'us' },
  { label: '일본', value: 'jp' }
];

const tags = [
  { label: 'Vue.js', value: 'vue' },
  { label: 'TypeScript', value: 'ts' },
  { label: 'JavaScript', value: 'js' }
];

const users = ref([]);

const handleUserSearch = async (query: string) => {
  // 사용자 검색 API 호출
  users.value = await searchUsers(query);
};
</script>
```

### 커스텀 옵션 렌더링

```vue
<template>
  <AppSelect
    v-model:value="selectedProduct"
    :options="products"
    placeholder="제품을 선택하세요"
  >
    <template #option="{ option, index }">
      <div class="flex items-center gap-3">
        <img :src="option.image" :alt="option.label" class="w-8 h-8 rounded" />
        <div>
          <div class="font-medium">{{ option.label }}</div>
          <div class="text-sm text-gray-500">{{ option.price }}</div>
        </div>
      </div>
    </template>
    
    <template #empty>
      <div class="text-center py-4">
        <p class="text-gray-500">제품이 없습니다</p>
        <AppButton size="small" @click="loadProducts">
          제품 불러오기
        </AppButton>
      </div>
    </template>
  </AppSelect>
</template>
```

## AppFormField

### 기본 사용법

```vue
<template>
  <form @submit.prevent="handleSubmit">
    <AppFormField
      label="이메일"
      :required="true"
      :status="emailStatus"
      :feedback="emailFeedback"
      help="로그인에 사용될 이메일 주소입니다"
    >
      <AppInput
        v-model:value="email"
        type="email"
        placeholder="이메일을 입력하세요"
        @blur="validateEmail"
      />
    </AppFormField>
    
    <AppFormField
      label="비밀번호"
      :required="true"
      :status="passwordStatus"
      :feedback="passwordFeedback"
    >
      <AppInput
        v-model:value="password"
        type="password"
        placeholder="비밀번호를 입력하세요"
        @input="validatePassword"
      />
    </AppFormField>
    
    <AppButton type="primary" html-type="submit" :disabled="!isFormValid">
      가입하기
    </AppButton>
  </form>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { AppFormField, AppInput, AppButton } from '@/components/ui/form';

const email = ref('');
const password = ref('');
const emailStatus = ref('default');
const passwordStatus = ref('default');
const emailFeedback = ref('');
const passwordFeedback = ref('');

const isFormValid = computed(() => {
  return emailStatus.value === 'success' && passwordStatus.value === 'success';
});

const validateEmail = () => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!email.value) {
    emailStatus.value = 'error';
    emailFeedback.value = '이메일을 입력해주세요';
  } else if (!emailRegex.test(email.value)) {
    emailStatus.value = 'error';
    emailFeedback.value = '올바른 이메일 형식이 아닙니다';
  } else {
    emailStatus.value = 'success';
    emailFeedback.value = '사용 가능한 이메일입니다';
  }
};

const validatePassword = () => {
  if (!password.value) {
    passwordStatus.value = 'default';
    passwordFeedback.value = '';
  } else if (password.value.length < 8) {
    passwordStatus.value = 'error';
    passwordFeedback.value = '비밀번호는 8자 이상이어야 합니다';
  } else {
    passwordStatus.value = 'success';
    passwordFeedback.value = '안전한 비밀번호입니다';
  }
};
</script>
```

### 레이아웃 옵션

```vue
<template>
  <!-- 라벨이 위에 (기본값) -->
  <AppFormField label="이름" label-placement="top">
    <AppInput v-model:value="name" />
  </AppFormField>
  
  <!-- 라벨이 왼쪽에 -->
  <AppFormField label="성별" label-placement="left">
    <AppSelect v-model:value="gender" :options="genderOptions" />
  </AppFormField>
</template>
```

## 접근성 가이드

### 키보드 네비게이션

모든 폼 컴포넌트는 완전한 키보드 네비게이션을 지원합니다:

- **Tab/Shift+Tab**: 필드 간 이동
- **Enter/Space**: 버튼 활성화
- **Escape**: 모달/드롭다운 닫기, 입력 내용 클리어
- **Arrow Keys**: 드롭다운 옵션 네비게이션

### ARIA 지원

```vue
<template>
  <!-- 스크린 리더를 위한 라벨링 -->
  <AppFormField
    label="비밀번호"
    :required="true"
    :status="passwordStatus"
    :feedback="passwordFeedback"
  >
    <AppInput
      v-model:value="password"
      type="password"
      aria-label="사용자 계정 비밀번호"
      aria-describedby="password-help"
      :required="true"
    />
  </AppFormField>
  
  <div id="password-help" class="text-sm text-gray-600">
    8자 이상의 영문, 숫자, 특수문자를 포함해야 합니다.
  </div>
</template>
```

### 에러 상태 처리

```vue
<template>
  <!-- 에러 메시지는 즉시 스크린 리더에 알림 -->
  <AppFormField
    label="이메일"
    :status="hasError ? 'error' : 'default'"
    :feedback="errorMessage"
    :required="true"
  >
    <AppInput
      v-model:value="email"
      type="email"
      :aria-invalid="hasError"
      aria-describedby="email-error"
    />
  </AppFormField>
</template>
```

## 테마 및 커스터마이징

### CSS Variables

컴포넌트는 CSS Variables를 통해 쉽게 커스터마이징할 수 있습니다:

```css
:root {
  /* Primary Colors */
  --primary-50: #eff6ff;
  --primary-500: #3b82f6;
  --primary-600: #2563eb;
  --primary-700: #1d4ed8;
  
  /* Status Colors */
  --success-500: #10b981;
  --warning-500: #f59e0b;
  --error-500: #ef4444;
  
  /* Border Radius */
  --border-radius-sm: 0.375rem;
  --border-radius-md: 0.5rem;
  --border-radius-lg: 0.75rem;
  
  /* Spacing */
  --spacing-xs: 0.5rem;
  --spacing-sm: 0.75rem;
  --spacing-md: 1rem;
  --spacing-lg: 1.5rem;
}
```

### 다크 모드

모든 컴포넌트는 자동으로 다크 모드를 지원합니다:

```vue
<template>
  <div class="dark">
    <!-- 다크 모드에서 자동으로 색상 변경 -->
    <AppButton type="primary">다크 모드 버튼</AppButton>
    <AppInput v-model:value="text" placeholder="다크 모드 입력" />
  </div>
</template>
```

### 커스텀 스타일

```vue
<template>
  <!-- 커스텀 클래스 적용 -->
  <AppButton class="custom-button" type="primary">
    커스텀 버튼
  </AppButton>
</template>

<style scoped>
.custom-button {
  --primary-500: #8b5cf6; /* 보라색으로 변경 */
  border-radius: 9999px; /* 완전히 둥근 모서리 */
  text-transform: uppercase; /* 대문자 변환 */
}
</style>
```

## 모범 사례

### 1. 폼 검증

```vue
<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAriaLive } from '@/composables/useAriaLive';

const { announceError, announceSuccess } = useAriaLive();

const validateForm = () => {
  const errors = [];
  
  if (!email.value) {
    errors.push('이메일을 입력해주세요');
  }
  
  if (errors.length > 0) {
    announceError(`${errors.length}개의 오류가 있습니다`);
    return false;
  }
  
  announceSuccess('폼이 성공적으로 제출되었습니다');
  return true;
};
</script>
```

### 2. 성능 최적화

```vue
<script setup lang="ts">
import { ref, watchDebounced } from '@vueuse/core';

const searchQuery = ref('');
const searchResults = ref([]);

// 디바운스를 통한 API 호출 최적화
watchDebounced(
  searchQuery,
  async (newQuery) => {
    if (newQuery.length > 2) {
      searchResults.value = await searchAPI(newQuery);
    }
  },
  { debounce: 300 }
);
</script>
```

### 3. 타입 안전성

```typescript
// types/form.ts
export interface UserFormData {
  name: string;
  email: string;
  age: number;
  country: string;
}

export interface FormFieldProps {
  label: string;
  required?: boolean;
  helpText?: string;
}
```

```vue
<script setup lang="ts">
import type { UserFormData } from '@/types/form';

const formData = ref<UserFormData>({
  name: '',
  email: '',
  age: 0,
  country: ''
});

const handleSubmit = async (data: UserFormData) => {
  // 타입 안전한 폼 처리
};
</script>
```

## 트러블슈팅

### 자주 발생하는 문제

1. **v-model이 작동하지 않을 때**
   ```vue
   <!-- 잘못된 사용 -->
   <AppInput v-model="value" />
   
   <!-- 올바른 사용 -->
   <AppInput v-model:value="value" />
   ```

2. **Select 옵션이 표시되지 않을 때**
   ```vue
   <!-- options 배열이 올바른 형태인지 확인 -->
   <AppSelect :options="[
     { label: '표시될 텍스트', value: '실제 값' }
   ]" />
   ```

3. **접근성 경고가 발생할 때**
   ```vue
   <!-- 아이콘 전용 버튼에는 반드시 aria-label 제공 -->
   <AppButton :circle="true" aria-label="설정 메뉴">
     <template #icon><SettingsIcon /></template>
   </AppButton>
   ```

### 디버깅 팁

1. **Vue DevTools 사용**: 컴포넌트 상태와 props 확인
2. **브라우저 접근성 도구**: WAVE, axe DevTools 활용
3. **키보드 네비게이션 테스트**: 마우스 없이 폼 사용해보기
4. **스크린 리더 테스트**: NVDA, JAWS 등으로 테스트

## 추가 리소스

- [Storybook 문서](./storybook-guide.md)
- [접근성 체크리스트](./accessibility-checklist.md)
- [디자인 토큰](./design-tokens.md)
- [컴포넌트 API 레퍼런스](./api-reference.md)