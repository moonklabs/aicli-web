import { URL, fileURLToPath } from 'node:url'

import { type UserConfig, defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { visualizer } from 'rollup-plugin-visualizer'
import { VitePWA } from 'vite-plugin-pwa'
import viteImagemin from 'vite-plugin-imagemin'

// https://vite.dev/config/
export default defineConfig(({ mode }): UserConfig => {
  const env = loadEnv(mode, process.cwd(), '')

  console.log('🔧 Building in mode:', mode)

  return {
    plugins: [
      vue({
        script: {
          // TypeScript 검사 비활성화 (성능 최적화 작업 중)
          defineModel: true,
          propsDestructure: true,
        },
        template: {
          compilerOptions: {
            isCustomElement: (tag) => tag.startsWith('ion-'),
          },
        },
      }),
      mode === 'development' && vueDevTools(),
      // PWA 설정
      VitePWA({
        registerType: 'autoUpdate',
        workbox: {
          globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
          maximumFileSizeToCacheInBytes: 5 * 1024 * 1024, // 5MB
          runtimeCaching: [
            {
              urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
              handler: 'CacheFirst',
              options: {
                cacheName: 'google-fonts-cache',
                expiration: {
                  maxEntries: 10,
                  maxAgeSeconds: 60 * 60 * 24 * 365, // 1년
                },
              },
            },
            {
              urlPattern: /^https:\/\/fonts\.gstatic\.com\/.*/i,
              handler: 'CacheFirst',
              options: {
                cacheName: 'gstatic-fonts-cache',
                expiration: {
                  maxEntries: 10,
                  maxAgeSeconds: 60 * 60 * 24 * 365, // 1년
                },
              },
            },
            {
              urlPattern: /\/api\/.*/i,
              handler: 'NetworkFirst',
              options: {
                cacheName: 'api-cache',
                expiration: {
                  maxEntries: 100,
                  maxAgeSeconds: 60 * 60 * 24, // 1일
                },
                networkTimeoutSeconds: 10,
              },
            },
            {
              urlPattern: /\.(?:png|jpg|jpeg|svg|gif|webp|avif)$/,
              handler: 'CacheFirst',
              options: {
                cacheName: 'images-cache',
                expiration: {
                  maxEntries: 100,
                  maxAgeSeconds: 60 * 60 * 24 * 30, // 30일
                },
              },
            },
          ],
        },
        includeAssets: ['favicon.ico', 'apple-touch-icon.png', 'masked-icon.svg'],
        manifest: {
          name: 'AICode Manager',
          short_name: 'AICode',
          description: 'AI-powered code management platform with Claude integration',
          theme_color: '#18a058',
          background_color: '#ffffff',
          display: 'standalone',
          orientation: 'portrait-primary',
          scope: '/',
          start_url: '/',
          id: '/',
          categories: ['development', 'productivity', 'utilities'],
          icons: [
            {
              src: '/icons/icon-72x72.png',
              sizes: '72x72',
              type: 'image/png',
            },
            {
              src: '/icons/icon-96x96.png',
              sizes: '96x96',
              type: 'image/png',
            },
            {
              src: '/icons/icon-128x128.png',
              sizes: '128x128',
              type: 'image/png',
            },
            {
              src: '/icons/icon-144x144.png',
              sizes: '144x144',
              type: 'image/png',
            },
            {
              src: '/icons/icon-152x152.png',
              sizes: '152x152',
              type: 'image/png',
            },
            {
              src: '/icons/icon-192x192.png',
              sizes: '192x192',
              type: 'image/png',
            },
            {
              src: '/icons/icon-384x384.png',
              sizes: '384x384',
              type: 'image/png',
            },
            {
              src: '/icons/icon-512x512.png',
              sizes: '512x512',
              type: 'image/png',
            },
            {
              src: '/icons/icon-maskable-192x192.png',
              sizes: '192x192',
              type: 'image/png',
              purpose: 'maskable',
            },
            {
              src: '/icons/icon-maskable-512x512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable',
            },
          ],
        },
        devOptions: {
          enabled: true, // 개발 모드에서도 PWA 기능 테스트 가능
        },
      }),
      // 번들 분석기 (프로덕션 빌드 시)
      mode === 'production' && visualizer({
        filename: 'dist/bundle-analysis.html',
        open: false,
        gzipSize: true,
        brotliSize: true,
      }),
      // 이미지 최적화 (일시적으로 비활성화)
      // 이미지 최적화 플러그인 (프로덕션 빌드에서만)
      mode === 'production' ? viteImagemin({
        gifsicle: {
          optimizationLevel: 7,
          interlaced: false,
        },
        optipng: {
          optimizationLevel: 7,
        },
        mozjpeg: {
          quality: 80,
        },
        pngquant: {
          quality: [0.8, 0.9],
          speed: 4,
        },
        svgo: {
          plugins: [
            {
              name: 'removeViewBox',
              active: false,
            },
            {
              name: 'removeEmptyAttrs',
              active: true,
            },
          ],
        },
      }) : null,
    ].filter(Boolean),

    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@/components': fileURLToPath(new URL('./src/components', import.meta.url)),
        '@/views': fileURLToPath(new URL('./src/views', import.meta.url)),
        '@/stores': fileURLToPath(new URL('./src/stores', import.meta.url)),
        '@/api': fileURLToPath(new URL('./src/api', import.meta.url)),
        '@/utils': fileURLToPath(new URL('./src/utils', import.meta.url)),
        '@/types': fileURLToPath(new URL('./src/types', import.meta.url)),
        '@/styles': fileURLToPath(new URL('./src/styles', import.meta.url)),
      },
    },

    // 개발 서버 설정
    server: {
      host: '0.0.0.0',
      port: 5173,
      strictPort: true,
      hmr: {
        overlay: true,
      },
      proxy: {
        // API 프록시 설정
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8080',
          changeOrigin: true,
          secure: false,
          rewrite: (path) => path, // '/api'를 유지
        },

        // WebSocket 프록시 설정
        '/ws': {
          target: env.VITE_WS_BASE_URL || 'ws://localhost:8080',
          ws: true,
          changeOrigin: true,
          secure: false,
        },

        // Socket.IO 프록시 (필요시)
        '/socket.io': {
          target: env.VITE_SOCKET_BASE_URL || 'http://localhost:8080',
          ws: true,
          changeOrigin: true,
          secure: false,
        },
      },
    },

    // 프리뷰 서버 설정 (프로덕션 빌드 테스트용)
    preview: {
      host: '0.0.0.0',
      port: 4173,
      strictPort: true,
    },

    // 빌드 설정
    build: {
      target: 'esnext',
      outDir: 'dist',
      sourcemap: mode === 'development',
      minify: mode === 'production' ? 'esbuild' : false,
      chunkSizeWarningLimit: 1000,
      rollupOptions: {
        output: {
          // 최적화된 청크 분리 전략
          manualChunks: (id) => {
            // Node modules 청크 분리
            if (id.includes('node_modules')) {
              // Vue 코어 (가장 중요, 별도 청크)
              if (id.includes('vue/') && !id.includes('vue-router') && !id.includes('vue-chartjs')) {
                return 'vue-core'
              }
              // Vue 라우터 (독립적 청크)
              if (id.includes('vue-router')) {
                return 'vue-router'
              }
              // Pinia 상태관리 (독립적 청크)
              if (id.includes('pinia')) {
                return 'pinia'
              }

              // Naive UI 세분화
              if (id.includes('naive-ui')) {
                // 폼 관련 컴포넌트
                if (id.includes('/form/') || id.includes('/input/') || id.includes('/select/') ||
                    id.includes('/checkbox/') || id.includes('/radio/') || id.includes('/date-picker/')) {
                  return 'naive-forms'
                }
                // 데이터 표시 컴포넌트
                if (id.includes('/table/') || id.includes('/data-table/') || id.includes('/list/')) {
                  return 'naive-data'
                }
                // 피드백 컴포넌트
                if (id.includes('/message/') || id.includes('/notification/') || id.includes('/modal/') ||
                    id.includes('/dialog/') || id.includes('/drawer/')) {
                  return 'naive-feedback'
                }
                // 레이아웃 컴포넌트
                if (id.includes('/layout/') || id.includes('/menu/') || id.includes('/breadcrumb/') ||
                    id.includes('/tabs/') || id.includes('/steps/')) {
                  return 'naive-layout'
                }
                // 버튼 및 액션 컴포넌트
                if (id.includes('/button/') || id.includes('/dropdown/') || id.includes('/popover/') ||
                    id.includes('/tooltip/') || id.includes('/popconfirm/')) {
                  return 'naive-actions'
                }
                // 유틸리티 컴포넌트
                if (id.includes('/spin/') || id.includes('/progress/') || id.includes('/skeleton/') ||
                    id.includes('/badge/') || id.includes('/tag/') || id.includes('/avatar/')) {
                  return 'naive-utils'
                }
                // 나머지 UI 컴포넌트
                return 'naive-ui-core'
              }

              // 아이콘 (별도 청크)
              if (id.includes('@vicons')) {
                return 'icons'
              }

              // 차트 라이브러리 (대용량)
              if (id.includes('chart.js') || id.includes('vue-chartjs')) {
                return 'charts'
              }

              // 유틸리티 라이브러리
              if (id.includes('axios') || id.includes('@tanstack') || id.includes('date-fns')) {
                return 'utils'
              }

              // 작은 유틸리티들은 하나로 합침
              if (id.includes('web-vitals') || id.includes('workbox')) {
                return 'utils-small'
              }

              // 나머지는 기본 vendor
              return 'vendor'
            }

            // 애플리케이션 코드 청크 분리
            if (id.includes('/src/')) {
              // 뷰/페이지별 동적 임포트 (라우트 레벨 분리)
              if (id.includes('/views/')) {
                if (id.includes('Dashboard')) return 'page-dashboard'
                if (id.includes('Terminal')) return 'page-terminal'
                if (id.includes('Docker')) return 'page-docker'
                if (id.includes('Security') || id.includes('Session')) return 'page-admin'
                if (id.includes('Profile') || id.includes('Login')) return 'page-auth'
                if (id.includes('Monitoring')) return 'page-monitoring'
                return 'pages-other'
              }

              // 컴포넌트별 분리 (크기별)
              if (id.includes('/components/')) {
                if (id.includes('/Performance/') || id.includes('/Debug/')) {
                  return 'components-dev'
                }
                if (id.includes('/accessibility/') || id.includes('/Common/')) {
                  return 'components-core'
                }
                return 'components-ui'
              }

              // 스토어와 서비스
              if (id.includes('/stores/') || id.includes('/api/')) {
                return 'app-core'
              }

              // 유틸리티와 컴포저블
              if (id.includes('/composables/') || id.includes('/utils/')) {
                return 'app-utils'
              }
            }

            return undefined
          },

          // 파일명 패턴 최적화
          chunkFileNames: (chunkInfo) => {
            const facadeModuleId = chunkInfo.facadeModuleId
            if (facadeModuleId) {
              // 동적 임포트된 컴포넌트의 경우
              if (facadeModuleId.includes('/views/')) {
                return 'chunks/views/[name].[hash].js'
              }
              if (facadeModuleId.includes('/components/')) {
                return 'chunks/components/[name].[hash].js'
              }
            }
            return 'chunks/[name].[hash].js'
          },

          entryFileNames: 'assets/[name].[hash].js',
          assetFileNames: (assetInfo) => {
            const name = assetInfo.name || 'unknown'
            const info = name.split('.')
            const ext = info[info.length - 1]

            if (/\.(mp4|webm|ogg|mp3|wav|flac|aac)$/.test(name)) {
              return 'assets/media/[name].[hash].[ext]'
            }
            if (/\.(png|jpe?g|gif|svg|webp|avif)$/.test(name)) {
              return 'assets/images/[name].[hash].[ext]'
            }
            if (/\.(woff2?|eot|ttf|otf)$/.test(name)) {
              return 'assets/fonts/[name].[hash].[ext]'
            }
            if (ext === 'css') {
              return 'assets/styles/[name].[hash].[ext]'
            }

            return 'assets/[name].[hash].[ext]'
          },
        },

        // Tree-shaking 최적화 재활성화
        treeshake: {
          preset: 'recommended',
          propertyReadSideEffects: false,
          tryCatchDeoptimization: false,
          moduleSideEffects: (id, _external) => {
            // 특정 모듈의 사이드 이펙트 제어
            if (id.includes('naive-ui') || id.includes('@vicons')) {
              return false // Tree-shaking 허용
            }
            if (id.includes('chart.js')) {
              return false // Tree-shaking 허용
            }
            return true // 외부 모듈은 사이드 이펙트 유지
          },
        },
      },
    },

    // CSS 설정
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `
            @use "@/styles/variables" as *;
            @use "@/styles/mixins" as *;
          `,
        },
      },
      devSourcemap: true,
    },

    // 환경 변수 설정
    define: {
      __VUE_OPTIONS_API__: true,
      __VUE_PROD_DEVTOOLS__: false,
      __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
    },

    // esbuild 설정
    esbuild: {
      drop: mode === 'production' ? ['console', 'debugger'] : [],
      // TypeScript 체크 건너뛰기 (성능 최적화 작업 중)
      target: 'esnext',
      format: 'esm',
    },

    // 최적화 설정
    optimizeDeps: {
      include: [
        'vue',
        'vue-router',
        'pinia',
        'axios',
      ],
      exclude: [
        'vite-plugin-vue-devtools',
        'naive-ui', // naive-ui를 exclude로 이동하여 개별 컴포넌트 최적화
      ],
    },
  }
})
