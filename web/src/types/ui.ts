/**
 * UI 컴포넌트 공통 타입 정의
 */

// 기본 사이즈 타입
export type Size = 'small' | 'medium' | 'large';

// 색상 변형 타입
export type ColorVariant = 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info';

// 컴포넌트 상태 타입
export type ComponentStatus = 'loading' | 'success' | 'error' | 'warning' | 'idle';

// 테마 모드 타입
export type ThemeMode = 'light' | 'dark' | 'auto';

// 방향 타입
export type Direction = 'horizontal' | 'vertical';

// 위치 타입
export type Position = 'top' | 'bottom' | 'left' | 'right';

// 정렬 타입
export type Alignment = 'start' | 'center' | 'end' | 'stretch';

// 간격 타입
export type Spacing = 'none' | 'small' | 'medium' | 'large' | 'xlarge';

// 버튼 타입
export interface ButtonProps {
  type?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  size?: Size;
  variant?: 'solid' | 'outline' | 'ghost' | 'text';
  disabled?: boolean;
  loading?: boolean;
  block?: boolean;
  round?: boolean;
  circle?: boolean;
  onClick?: (event: Event) => void;
  class?: string;
  style?: string | Record<string, any>;
}

// 입력 컴포넌트 기본 속성
export interface InputBaseProps {
  value?: string | number;
  placeholder?: string;
  disabled?: boolean;
  readonly?: boolean;
  size?: Size;
  status?: 'default' | 'success' | 'warning' | 'error';
  clearable?: boolean;
  maxlength?: number;
  showCount?: boolean;
  onUpdate:value?: (value: string | number) => void;
  onFocus?: (event: FocusEvent) => void;
  onBlur?: (event: FocusEvent) => void;
  onChange?: (value: string | number) => void;
}

// 폼 필드 속성
export interface FormFieldProps {
  label?: string;
  required?: boolean;
  feedback?: string;
  status?: ComponentStatus;
  showFeedback?: boolean;
  labelPlacement?: 'top' | 'left';
  feedbackPlacement?: 'bottom' | 'right';
}

// 로딩 상태 속성
export interface LoadingProps {
  show?: boolean;
  size?: Size;
  description?: string;
  delay?: number;
}

// 스켈레톤 속성
export interface SkeletonProps {
  text?: boolean;
  round?: boolean;
  circle?: boolean;
  height?: string | number;
  width?: string | number;
  repeat?: number;
  animated?: boolean;
}

// 에러 상태 속성
export interface ErrorStateProps {
  title?: string;
  description?: string;
  status?: number;
  showRetry?: boolean;
  onRetry?: () => void;
}

// 빈 상태 속성
export interface EmptyStateProps {
  description?: string;
  size?: Size;
  showIcon?: boolean;
  iconName?: string;
}

// 모달 속성
export interface ModalProps {
  show?: boolean;
  title?: string;
  closable?: boolean;
  maskClosable?: boolean;
  loading?: boolean;
  autoFocus?: boolean;
  trapFocus?: boolean;
  blockScroll?: boolean;
  size?: Size | 'small' | 'medium' | 'large' | 'huge';
  onClose?: () => void;
  onAfterEnter?: () => void;
  onAfterLeave?: () => void;
}

// 데이터 테이블 컬럼 타입
export interface TableColumn {
  key: string;
  title: string;
  width?: number | string;
  minWidth?: number | string;
  maxWidth?: number | string;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  filterable?: boolean;
  fixed?: 'left' | 'right';
  ellipsis?: boolean;
  render?: (row: any, index: number) => any;
  renderHeader?: () => any;
}

// 데이터 테이블 속성
export interface DataTableProps {
  data?: any[];
  columns?: TableColumn[];
  loading?: boolean;
  pagination?: boolean;
  pageSize?: number;
  scrollX?: number | string;
  scrollY?: number | string;
  striped?: boolean;
  bordered?: boolean;
  singleLine?: boolean;
  size?: Size;
  rowKey?: string | ((row: any) => string | number);
  rowClassName?: string | ((row: any, index: number) => string);
  onUpdateCheckedRowKeys?: (keys: Array<string | number>) => void;
  onUpdatePage?: (page: number) => void;
  onUpdatePageSize?: (pageSize: number) => void;
  onUpdateSorter?: (sorter: any) => void;
  onUpdateFilters?: (filters: any) => void;
}

// 테마 구성 타입
export interface ThemeConfig {
  mode: ThemeMode;
  primaryColor?: string;
  borderRadius?: string;
  fontSize?: string;
  fontFamily?: string;
}

// 접근성 속성
export interface AccessibilityProps {
  'aria-label'?: string;
  'aria-labelledby'?: string;
  'aria-describedby'?: string;
  'aria-hidden'?: boolean;
  'aria-expanded'?: boolean;
  'aria-disabled'?: boolean;
  'aria-required'?: boolean;
  'aria-invalid'?: boolean;
  'aria-live'?: 'off' | 'polite' | 'assertive';
  role?: string;
  tabindex?: number;
}

// 키보드 이벤트 속성
export interface KeyboardEventProps {
  onKeydown?: (event: KeyboardEvent) => void;
  onKeyup?: (event: KeyboardEvent) => void;
  onKeypress?: (event: KeyboardEvent) => void;
}

// 포커스 관리 속성
export interface FocusProps {
  autofocus?: boolean;
  tabindex?: number;
  onFocus?: (event: FocusEvent) => void;
  onBlur?: (event: FocusEvent) => void;
}

// 반응형 속성
export interface ResponsiveProps {
  xs?: any;
  sm?: any;
  md?: any;
  lg?: any;
  xl?: any;
  xxl?: any;
}

// 애니메이션 속성
export interface AnimationProps {
  duration?: number;
  easing?: string;
  delay?: number;
}

// 기본 컴포넌트 속성 (모든 컴포넌트가 상속)
export interface BaseComponentProps extends AccessibilityProps, KeyboardEventProps, FocusProps {
  id?: string;
  class?: string | string[] | Record<string, boolean>;
  style?: string | Record<string, any>;
  'data-testid'?: string;
}