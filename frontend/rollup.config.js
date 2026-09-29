import svelte from 'rollup-plugin-svelte';
import commonjs from '@rollup/plugin-commonjs';
import resolve from '@rollup/plugin-node-resolve';
import livereload from 'rollup-plugin-livereload';
import terser from '@rollup/plugin-terser';
import css from 'rollup-plugin-css-only';
import fs from 'fs';
import path from 'path';

const production = !process.env.ROLLUP_WATCH;

function copyPublic() {
	return {
		name: 'copy-public',
		buildStart() {
			const publicDir = 'public';
			const buildDir = 'build';
			
			if (!fs.existsSync(buildDir)) {
				fs.mkdirSync(buildDir, { recursive: true });
			}
			
			if (fs.existsSync(publicDir)) {
				copyDir(publicDir, buildDir);
			}
		}
	};
}

function copyDir(src, dest) {
	const entries = fs.readdirSync(src, { withFileTypes: true });
	
	for (const entry of entries) {
		const srcPath = path.join(src, entry.name);
		const destPath = path.join(dest, entry.name);
		
		if (entry.isDirectory()) {
			if (!fs.existsSync(destPath)) {
				fs.mkdirSync(destPath, { recursive: true });
			}
			copyDir(srcPath, destPath);
		} else {
			fs.copyFileSync(srcPath, destPath);
		}
	}
}

export default {
	input: 'src/main.js',
	output: {
		sourcemap: true,
		format: 'iife',
		name: 'app',
		file: 'build/bundle.js'
	},
	plugins: [
		svelte({
			compilerOptions: {
				// enable run-time checks when not in production
				dev: !production
			}
		}),
		// we'll extract any component CSS out into
		// a separate file - better for performance
		css({ output: 'bundle.css' }),

		// If you have external dependencies installed from
		// npm, you'll most likely need these plugins. In
		// some cases you'll need additional configuration -
		// consult the documentation for details:
		// https://github.com/rollup/plugins/tree/master/packages/commonjs
		resolve({
			browser: true,
			dedupe: ['svelte']
		}),
		commonjs(),

		// Copy public directory to build
		copyPublic(),

		// 开发模式下不自动启动本地静态服务器：静态文件由 Go 后端统一提供
		// （原 serve() 插件已移除，因为不再需要 npm run start）

		// Watch the `public` directory and refresh the
		// browser on changes when not in production
		!production && livereload('public'),

		// If we're building for production (npm run build
		// instead of npm run dev), minify
		production && terser()
	],
	watch: {
		clearScreen: false
	}
};
