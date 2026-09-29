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

// 色彩模式：light 亮色 / dark 暗色 / system 跟随系统（默认）
export const colorMode = createPersistedStore('colorMode', 'system');

export const COLOR_MODES = [
  { id: 'light', name: '亮色', icon: 'fas fa-sun' },
  { id: 'dark', name: '暗色', icon: 'fas fa-moon' },
  { id: 'system', name: '系统', icon: 'fas fa-circle-half-stroke' },
];

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
  const rgb = hexToRgb(hex);
  if (!rgb) {
    return hex;
  }
  return `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${alpha})`;
}

function hexToRgb(hex) {
  const raw = String(hex || '').trim().replace('#', '');
  const full = raw.length === 3 ? raw.split('').map((c) => c + c).join('') : raw;
  if (full.length !== 6 || /[^0-9a-fA-F]/.test(full)) {
    return null;
  }
  return [
    parseInt(full.slice(0, 2), 16),
    parseInt(full.slice(2, 4), 16),
    parseInt(full.slice(4, 6), 16),
  ];
}

function toHex(rgb) {
  return '#' + rgb.map((v) => Math.round(Math.min(255, Math.max(0, v))).toString(16).padStart(2, '0')).join('');
}

// mixHex 按比例混合两个颜色；weight 是第二个颜色的占比（0~1）
export function mixHex(a, b, weight = 0.5) {
  const ca = hexToRgb(a);
  const cb = hexToRgb(b);
  if (!ca || !cb) {
    return a;
  }
  const w = Math.min(1, Math.max(0, weight));
  return toHex([
    ca[0] * (1 - w) + cb[0] * w,
    ca[1] * (1 - w) + cb[1] * w,
    ca[2] * (1 - w) + cb[2] * w,
  ]);
}

// 亮色模式下强调色向主题深色靠拢的比例：越高越深、在浅底上越可读
const LIGHT_ACCENT_MIX = 0.62;

// getThemeTokens 把主题色值映射成语义 token，组件只使用 token。
//
// 关键约定：colors[0]/[1] 是「深色背景」，只能当背景；前景一律走 --ink 系，
// 强调色走 --accent。早期把深色背景色当字色用，导致选中态是「深字压深底」不可读。
//
// mode = light 时由主题色派生一套亮色 token（浅底 + 深字 + 加深的强调色），
// 6 个主题因此都自动支持亮色，无需为每个主题维护两套色板；
// 若某个主题需要更精细的亮色，后续可在 themes 里为它显式指定 lightColors 覆盖。
export function getThemeTokens(themeId, mode = 'dark') {
  const theme = themes.find(t => t.id === themeId) || themes[0];
  const [bgDeep, bgMid, accent] = theme.colors;

  if (mode === 'light') {
    const bgTop = mixHex(accent, '#ffffff', 0.9);
    const bgBottom = mixHex(accent, '#ffffff', 0.8);
    const text = mixHex(bgDeep, '#000000', 0.12);
    const accentStrong = mixHex(accent, bgDeep, LIGHT_ACCENT_MIX);

    return {
      '--bg-deep': bgTop,
      '--bg-mid': bgBottom,
      '--bg-grad': `linear-gradient(165deg, ${bgTop} 0%, ${bgBottom} 100%)`,

      '--surface': 'rgba(0, 0, 0, 0.04)',
      '--surface-hover': 'rgba(0, 0, 0, 0.08)',
      '--border': 'rgba(0, 0, 0, 0.12)',
      '--border-strong': 'rgba(0, 0, 0, 0.24)',

      '--ink': text,
      '--ink-muted': hexToRgba(text, 0.72),
      '--ink-soft': hexToRgba(text, 0.56),
      '--ink-faint': hexToRgba(text, 0.4),

      '--accent': accentStrong,
      '--accent-soft': hexToRgba(accentStrong, 0.14),
      '--accent-line': hexToRgba(accentStrong, 0.45),
      '--accent-ink': '#ffffff',

      '--ok-ink': '#1b5e20',
      '--warn-ink': '#7a5c00',
      '--danger-ink': '#b3261e',

      '--panel': 'rgba(255, 255, 255, 0.97)',
      '--panel-inset': 'rgba(0, 0, 0, 0.05)',
      '--menu': 'rgba(255, 255, 255, 0.98)',
      '--overlay-strong': 'rgba(255, 255, 255, 0.86)',
      '--code-bg': 'rgba(0, 0, 0, 0.05)',
      '--scrim': 'rgba(15, 23, 42, 0.42)',
      '--shadow': 'rgba(15, 23, 42, 0.16)',
      '--shadow-strong': 'rgba(15, 23, 42, 0.28)',
    };
  }

  return {
    '--bg-deep': bgDeep,
    '--bg-mid': bgMid,
    '--bg-grad': `linear-gradient(165deg, ${bgDeep} 0%, ${bgMid} 100%)`,

    '--surface': 'rgba(255, 255, 255, 0.06)',
    '--surface-hover': 'rgba(255, 255, 255, 0.12)',
    '--border': 'rgba(255, 255, 255, 0.12)',
    '--border-strong': 'rgba(255, 255, 255, 0.26)',

    '--ink': '#ffffff',
    '--ink-muted': 'rgba(255, 255, 255, 0.7)',
    '--ink-soft': 'rgba(255, 255, 255, 0.52)',
    '--ink-faint': 'rgba(255, 255, 255, 0.38)',

    '--accent': accent,
    '--accent-soft': hexToRgba(accent, 0.16),
    '--accent-line': hexToRgba(accent, 0.55),
    // 压在强调色上的文字色：用深色背景色，保证对比度
    '--accent-ink': bgDeep,

    '--ok-ink': '#b9f6ca',
    '--warn-ink': '#ffe082',
    '--danger-ink': '#ff9aa2',

    '--panel': 'rgba(12, 18, 30, 0.97)',
    '--panel-inset': 'rgba(0, 0, 0, 0.35)',
    '--menu': 'rgba(24, 30, 46, 0.96)',
    '--overlay-strong': 'rgba(10, 16, 28, 0.75)',
    '--code-bg': 'rgba(0, 0, 0, 0.3)',
    '--scrim': 'rgba(0, 0, 0, 0.6)',
    '--shadow': 'rgba(0, 0, 0, 0.35)',
    '--shadow-strong': 'rgba(0, 0, 0, 0.5)',
  };
}

// themeSwatch 主题选择器里的预览色块：跟随当前色彩模式，避免亮色模式下预览仍是深色
export function themeSwatch(themeId, mode = 'dark') {
  const tokens = getThemeTokens(themeId, mode);
  return `linear-gradient(135deg, ${tokens['--bg-deep']}, ${tokens['--accent']})`;
}
