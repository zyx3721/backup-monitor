import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { nitro } from 'nitro/vite';
import path from 'path';
import { defineConfig, loadEnv } from 'vite';
import tsconfigPaths from 'vite-tsconfig-paths';

export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiTarget = env.VITE_API_BASE_URL || 'http://127.0.0.1:8080';

  return {
    plugins: [
      tailwindcss(),
      tsconfigPaths({ projects: ['./tsconfig.json'] }),
      tanstackStart({
        importProtection: {
          behavior: 'error',
          client: {
            files: ['**/server/**'],
            specifiers: ['server-only'],
          },
        },
        server: { entry: 'server' },
      }),
      ...(command === 'build'
        ? nitro({
            rollupConfig: {
              onwarn(warning, warn) {
                if (warning.code === 'MODULE_LEVEL_DIRECTIVE') return;
                if (
                  warning.code &&
                  ['EVAL', 'CIRCULAR_DEPENDENCY', 'THIS_IS_UNDEFINED', 'EMPTY_BUNDLE'].includes(
                    warning.code
                  )
                )
                  return;
                warn(warning);
              },
            },
          })
        : []),
      react(),
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
          ws: true,
        },
      },
    },
  };
});
