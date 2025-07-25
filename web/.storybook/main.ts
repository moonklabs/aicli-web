import type { StorybookConfig } from '@storybook/vue3-vite';

const config: StorybookConfig = {
  stories: [
    "../src/**/*.mdx",
    "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"
  ],
  addons: [
    "@storybook/addon-docs",
    "@storybook/addon-onboarding"
  ],
  framework: {
    name: "@storybook/vue3-vite",
    options: {}
  },
  viteFinal: async (config) => {
    // Vite 설정 커스터마이징
    config.define = {
      ...config.define,
      global: 'globalThis',
    };
    
    return config;
  }
};

export default config;