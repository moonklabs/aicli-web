import { URL, fileURLToPath } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { visualizer } from 'rollup-plugin-visualizer'
import { VitePWA } from 'vite-plugin-pwa'
// import { viteImagemin } from 'vite-plugin-imagemin'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      vue(),
      mode === 'development' ? vueDevTools() : null,
      // PWA 설정
      VitePWA({
        registerType: 'autoUpdate',
        workbox: {
          globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
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
      mode === 'production' ? visualizer({
        filename: 'dist/bundle-analysis.html',
        open: false,
        gzipSize: true,
        brotliSize: true,
      }) : null,
      // 이미지 최적화 (프로덕션 빌드 시) - 일시적으로 비활성화
      // mode === 'production' ? viteImagemin({
      //   gifsicle: { optimizationLevel: 7 },
      //   mozjpeg: { quality: 80 },
      //   pngquant: { quality: [0.65, 0.8] },
      //   webp: { quality: 80 },
      // }) : null,
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
          // 더 세밀한 청크 분리
          manualChunks: (id) => {
            // Node modules 청크 분리
            if (id.includes('node_modules')) {
              // Vue 생태계
              if (id.includes('vue') || id.includes('pinia') || id.includes('@vue')) {
                return 'vue-vendor'
              }
              // UI 라이브러리
              if (id.includes('naive-ui') || id.includes('@vicons')) {
                return 'ui-vendor'
              }
              // 차트 라이브러리
              if (id.includes('chart.js') || id.includes('vue-chartjs')) {
                return 'chart-vendor'
              }
              // HTTP 및 유틸리티
              if (id.includes('axios') || id.includes('@tanstack')) {
                return 'utils-vendor'
              }
              // 나머지 vendor
              return 'vendor'
            }

            // 컴포넌트 청크 분리
            if (id.includes('/src/components/')) {
              if (id.includes('/forms/')) {
                return 'forms-components'
              }
              if (id.includes('/overlay/')) {
                return 'overlay-components'
              }
              if (id.includes('/charts/')) {
                return 'chart-components'
              }
              if (id.includes('/tables/')) {
                return 'table-components'
              }
              return 'ui-components'
            }

            // 뷰/페이지 청크 분리
            if (id.includes('/src/views/')) {
              return 'views'
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
            const info = assetInfo.name!.split('.')
            const ext = info[info.length - 1]

            if (/\.(mp4|webm|ogg|mp3|wav|flac|aac)$/.test(assetInfo.name!)) {
              return 'assets/media/[name].[hash].[ext]'
            }
            if (/\.(png|jpe?g|gif|svg|webp|avif)$/.test(assetInfo.name!)) {
              return 'assets/images/[name].[hash].[ext]'
            }
            if (/\.(woff2?|eot|ttf|otf)$/.test(assetInfo.name!)) {
              return 'assets/fonts/[name].[hash].[ext]'
            }
            if (ext === 'css') {
              return 'assets/styles/[name].[hash].[ext]'
            }

            return 'assets/[name].[hash].[ext]'
          },
        },

        // Tree-shaking 최적화
        treeshake: {
          moduleSideEffects: false,
          preset: 'recommended',
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
    },

    // 최적화 설정
    optimizeDeps: {
      include: [
        'vue',
        'vue-router',
        'pinia',
        'axios',
        'naive-ui',
      ],
      exclude: [
        'vite-plugin-vue-devtools',
      ],
    },
  }
})
