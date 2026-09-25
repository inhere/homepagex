import { writable } from 'svelte/store';

export const pageConfig = writable({});
export const currentRoute = writable('/');
// 当前登录用户信息，null 表示游客
export const userInfo = writable(null);

function createPersistedStore(key, initialValue) {
  const storedValue = localStorage.getItem(key);
  const initial = storedValue ? JSON.parse(storedValue) : initialValue;
  const store = writable(initial);

  store.subscribe(value => {
    localStorage.setItem(key, JSON.stringify(value));
  });

  return store;
}

export const viewStyle = createPersistedStore('viewStyle', 'cards');

export const themes = [
  {
    id: 'ocean-depths',
    name: '海洋深处',
    colors: ['#1a2332', '#12293a', '#a8dadc', '#f1faee'],
    description: '专业宁静的海洋主题'
  },
  {
    id: 'tech-innovation',
    name: '科技创新',
    colors: ['#1e1e1e', '#0b2a4a', '#00ffff', '#ffffff'],
    description: '科技感十足的现代主题'
  },
  {
    id: 'modern-minimalist',
    name: '现代极简',
    colors: ['#36454f', '#26333b', '#d3d3d3', '#ffffff'],
    description: '简洁专业的极简主义'
  },
  {
    id: 'midnight-galaxy',
    name: '午夜星河',
    colors: ['#2b1e3e', '#1b1430', '#a490c2', '#e6e6fa'],
    description: '深邃神秘的宇宙主题'
  },
  {
    id: 'forest-canopy',
    name: '森林树冠',
    colors: ['#2d4a2b', '#1d3320', '#a4ac86', '#faf9f6'],
    description: '自然清新的森林主题'
  },
  {
    id: 'arctic-frost',
    name: '北极冰霜',
    colors: ['#3b5a86', '#2a4368', '#c0c0c0', '#fafafa'],
    description: '清爽现代的冰雪主题'
  },
];

export const currentTheme = createPersistedStore('theme', 'ocean-depths');

// hexToRgba 把 #rgb / #rrggbb 转成 rgba()。
// 不再用「色值 + "33"」这种字符串拼接 —— 那只对 6 位 hex 成立，
// 一旦写成 #abc / rgb() / hsl() 就会产出非法 CSS 并被整条丢弃。
export function hexToRgba(hex, alpha = 1) {
  const raw = String(hex || '').trim().replace('#', '');
  let full = raw;
  if (raw.length === 3) {
    full = raw.split('').map((c) => c + c).join('');
  }
  if (full.length !== 6 || /[^0-9a-fA-F]/.test(full)) {
    return hex;
  }
  const r = parseInt(full.slice(0, 2), 16);
  const g = parseInt(full.slice(2, 4), 16);
  const b = parseInt(full.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// getThemeTokens 把主题色值映射成语义 token。
//
// 关键点：colors[0]/colors[1] 是「深色背景」，只能用作背景；
// 前景色一律用 --ink 系（白），强调色一律用 --accent（colors[2]，浅色）。
// 之前把 --theme-primary（深色背景色）当前景色用，导致选中态是「深字压深底」不可读。
export function getThemeTokens(themeId) {
  const theme = themes.find(t => t.id === themeId) || themes[0];
  const bgDeep = theme.colors[0];
  const bgMid = theme.colors[1];
  const accent = theme.colors[2];

  return {
    '--bg-deep': bgDeep,
    '--bg-mid': bgMid,
    '--bg-grad': `linear-gradient(165deg, ${bgDeep} 0%, ${bgMid} 100%)`,

    '--surface': 'rgba(255, 255, 255, 0.06)',
    '--surface-hover': 'rgba(255, 255, 255, 0.12)',
    '--border': 'rgba(255, 255, 255, 0.12)',
    '--border-strong': 'rgba(255, 255, 255, 0.24)',

    '--ink': '#ffffff',
    '--ink-muted': 'rgba(255, 255, 255, 0.68)',
    '--ink-soft': 'rgba(255, 255, 255, 0.45)',

    '--accent': accent,
    '--accent-soft': hexToRgba(accent, 0.16),
    '--accent-line': hexToRgba(accent, 0.55),
    // 压在强调色上的文字色：用深色背景色，保证对比度
    '--accent-ink': bgDeep,
  };
}
