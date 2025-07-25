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

// 데이터 테이블 고급 필터 타입
export interface TableFilter {
  key: string;
  value: any;
  operator?: 'equals' | 'contains' | 'startsWith' | 'endsWith' | 'gt' | 'gte' | 'lt' | 'lte' | 'between' | 'in' | 'notIn';
  type?: 'text' | 'number' | 'date' | 'select' | 'boolean';
}

// 데이터 테이블 정렬 타입
export interface TableSorter {
  key: string;
  order: 'asc' | 'desc';
  sorter?: 'default' | 'alphanumeric' | 'numeric' | 'date' | ((a: any, b: any) => number);
}

// 가상 스크롤링 설정
export interface VirtualScrollConfig {
  enabled?: boolean;
  itemHeight?: number | 'auto';
  overscan?: number;
  scrollContainer?: string | HTMLElement;
}

// 데이터 테이블 페이지네이션 설정
export interface TablePagination {
  page: number;
  pageSize: number;
  total: number;
  showSizeChanger?: boolean;
  pageSizes?: number[];
  showQuickJumper?: boolean;
  position?: 'top' | 'bottom' | 'both';
}

// 확장된 데이터 테이블 컬럼 타입
export interface AdvancedTableColumn extends TableColumn {
  resizable?: boolean;
  hideable?: boolean;
  pinnable?: boolean;
  groupable?: boolean;
  searchable?: boolean;
  filter?: {
    type: 'text' | 'number' | 'date' | 'select' | 'multiSelect' | 'dateRange';
    options?: Array<{ label: string; value: any }>;
    placeholder?: string;
    multiple?: boolean;
  };
  sort?: {
    compare?: (a: any, b: any) => number;
    multiple?: boolean;
  };
  export?: {
    exclude?: boolean;
    formatter?: (value: any) => string;
  };
}

// 고급 데이터 테이블 속성
export interface AdvancedDataTableProps extends Omit<DataTableProps, 'columns'> {
  columns?: AdvancedTableColumn[];
  virtualScroll?: VirtualScrollConfig;
  filters?: TableFilter[];
  sorters?: TableSorter[];
  selection?: {
    type?: 'checkbox' | 'radio' | 'none';
    selectedKeys?: Array<string | number>;
    onSelectionChange?: (keys: Array<string | number>) => void;
  };
  export?: {
    enabled?: boolean;
    formats?: Array<'csv' | 'excel' | 'json'>;
    filename?: string;
  };
  grouping?: {
    enabled?: boolean;
    groupBy?: string[];
    expanded?: string[];
  };
  responsive?: {
    enabled?: boolean;
    breakpoints?: Record<string, number>;
    hideColumns?: Record<string, string[]>;
  };
  performance?: {
    debounceMs?: number;
    throttleMs?: number;
    lazyLoading?: boolean;
  };
  onRowClick?: (row: any, index: number, event: Event) => void;
  onRowDoubleClick?: (row: any, index: number, event: Event) => void;
  onCellClick?: (cell: any, row: any, column: AdvancedTableColumn, event: Event) => void;
}

// 차트 데이터 포인트 타입
export interface ChartDataPoint {
  x?: any;
  y?: any;
  label?: string;
  value?: number;
  color?: string;
  [key: string]: any;
}

// 차트 데이터셋 타입
export interface ChartDataset {
  label: string;
  data: ChartDataPoint[] | number[];
  backgroundColor?: string | string[];
  borderColor?: string | string[];
  borderWidth?: number;
  tension?: number;
  fill?: boolean;
  pointRadius?: number;
  pointHoverRadius?: number;
  [key: string]: any;
}

// 차트 데이터 구조
export interface ChartData {
  labels?: string[];
  datasets: ChartDataset[];
}

// 차트 기본 속성
export interface BaseChartProps extends BaseComponentProps {
  data: ChartData;
  options?: any;
  plugins?: any[];
  width?: number;
  height?: number;
  responsive?: boolean;
  maintainAspectRatio?: boolean;
  redraw?: boolean;
  fallbackContent?: string;
  onChartCreate?: (chart: any) => void;
  onChartUpdate?: (chart: any) => void;
  onChartDestroy?: (chart: any) => void;
  onClick?: (event: Event, elements: any[]) => void;
  onHover?: (event: Event, elements: any[]) => void;
}

// 라인 차트 속성
export interface LineChartProps extends BaseChartProps {
  stepped?: boolean | 'before' | 'after' | 'middle';
  spanGaps?: boolean | number;
  showLine?: boolean;
  tension?: number;
}

// 바 차트 속성
export interface BarChartProps extends BaseChartProps {
  indexAxis?: 'x' | 'y';
  skipNull?: boolean;
  grouped?: boolean;
  barPercentage?: number;
  categoryPercentage?: number;
}

// 파이/도넛 차트 속성
export interface PieChartProps extends BaseChartProps {
  cutout?: number | string;
  circumference?: number;
  rotation?: number;
  animations?: {
    animateRotate?: boolean;
    animateScale?: boolean;
  };
}

// 스캐터 차트 속성
export interface ScatterChartProps extends BaseChartProps {
  showLine?: boolean;
  pointRadius?: number;
  pointHoverRadius?: number;
}

// 차트 테마 설정
export interface ChartTheme {
  colors: {
    primary: string[];
    secondary: string[];
    accent: string[];
    neutral: string[];
  };
  fonts: {
    family: string;
    size: number;
    weight: string | number;
  };
  grid: {
    color: string;
    lineWidth: number;
  };
  tooltip: {
    backgroundColor: string;
    titleColor: string;
    bodyColor: string;
    borderColor: string;
  };
}

// 실시간 차트 업데이트 설정
export interface RealTimeChartConfig {
  enabled?: boolean;
  interval?: number;
  maxDataPoints?: number;
  animationDuration?: number;
  onDataUpdate?: (newData: any) => void;
}

// 차트 내보내기 설정
export interface ChartExportConfig {
  enabled?: boolean;
  formats?: Array<'png' | 'jpg' | 'svg' | 'pdf'>;
  quality?: number;
  backgroundColor?: string;
  filename?: string;
}

// 차트 줌/팬 설정
export interface ChartZoomConfig {
  enabled?: boolean;
  mode?: 'x' | 'y' | 'xy';
  rangeMin?: {
    x?: any;
    y?: any;
  };
  rangeMax?: {
    x?: any;
    y?: any;
  };
  speed?: number;
  threshold?: number;
  onZoomComplete?: (context: any) => void;
  onPanComplete?: (context: any) => void;
}

// 차트-테이블 연동 설정
export interface ChartTableIntegration {
  enabled?: boolean;
  syncSelection?: boolean;
  syncFiltering?: boolean;
  highlightOnHover?: boolean;
  onSelectionSync?: (selection: any) => void;
  onFilterSync?: (filters: TableFilter[]) => void;
}

// 고급 차트 위젯 속성
export interface AdvancedChartProps extends BaseChartProps {
  theme?: ChartTheme;
  realTime?: RealTimeChartConfig;
  export?: ChartExportConfig;
  zoom?: ChartZoomConfig;
  tableIntegration?: ChartTableIntegration;
  accessibility?: {
    enabled?: boolean;
    description?: string;
    summary?: string;
  };
  loading?: boolean;
  error?: Error | null;
  onError?: (error: Error) => void;
}

// 기본 컴포넌트 속성 (모든 컴포넌트가 상속)
export interface BaseComponentProps extends AccessibilityProps, KeyboardEventProps, FocusProps {
  id?: string;
  class?: string | string[] | Record<string, boolean>;
  style?: string | Record<string, any>;
  'data-testid'?: string;
}