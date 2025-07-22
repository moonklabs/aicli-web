import { URL, fileURLToPath } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      vue(),
      mode === 'development' ? vueDevTools() : null,
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
          manualChunks: {
            // 벤더 청크 분리
            vendor: ['vue', 'vue-router', 'pinia'],
            ui: ['naive-ui'],
            http: ['axios'],
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
