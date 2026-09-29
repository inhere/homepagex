import js from '@eslint/js';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';

export default [
  // build/ 是 rollup 产物（已在 .gitignore 中），public/build/ 同理
  { ignores: ['build/', 'public/build/'] },

  js.configs.recommended,
  ...svelte.configs['flat/recommended'],

  {
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        ...globals.browser,
      },
    },
    rules: {
      // 加这两条的直接原因：App.svelte 曾漏 import userInfo，
      // Svelte 编译期不报错、构建也通过，但运行时抛 ReferenceError。
      'no-undef': 'error',
      'no-unused-vars': ['warn', { args: 'none', ignoreRestSiblings: true }],
    },
  },

  {
    // rollup.config.js / eslint.config.js 这类跑在 Node 里的脚本
    files: ['**/*.config.js'],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
];
