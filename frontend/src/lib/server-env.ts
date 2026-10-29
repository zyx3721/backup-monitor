// serverEnv 供服务端渲染读取配置：优先运行时环境变量，其次入口文件所在目录向上各层的 .env 文件
let cachedFile: Record<string, string> | null | undefined;

function parseEnvFile(content: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const rawLine of content.split(/\r?\n/)) {
    let line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    if (line.startsWith('export ')) line = line.slice(7).trim();
    const eq = line.indexOf('=');
    if (eq <= 0) continue;
    const key = line.slice(0, eq).trim();
    let value = line.slice(eq + 1).trim();
    if (
      value.length >= 2 &&
      ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'")))
    ) {
      value = value.slice(1, -1);
    }
    if (key) out[key] = value;
  }
  return out;
}

export async function serverEnv(key: string): Promise<string | undefined> {
  if (typeof window !== 'undefined') return undefined;
  const proc = (
    globalThis as {
      process?: {
        env?: Record<string, string | undefined>;
        cwd?: () => string;
        argv?: string[];
      };
    }
  ).process;
  const fromProcess = proc?.env?.[key];
  if (fromProcess) return fromProcess;
  if (cachedFile === undefined) {
    cachedFile = null;
    try {
      const fsName = 'node:' + 'fs';
      const pathName = 'node:' + 'path';
      const [fs, path] = await Promise.all([
        import(/* @vite-ignore */ fsName),
        import(/* @vite-ignore */ pathName),
      ]);
      const seen = new Set<string>();
      const dirs: string[] = [];
      const entry = proc?.argv?.[1];
      if (entry) {
        let dir = path.dirname(path.resolve(entry));
        while (true) {
          dirs.push(dir);
          const parent = path.dirname(dir);
          if (parent === dir) break;
          dir = parent;
        }
      }
      const cwd = proc?.cwd?.();
      if (cwd) dirs.push(cwd);
      for (const dir of dirs) {
        const file = path.join(dir, '.env');
        if (seen.has(file)) continue;
        seen.add(file);
        try {
          cachedFile = parseEnvFile(fs.readFileSync(file, 'utf8'));
          break;
        } catch {
          cachedFile = null;
        }
      }
    } catch {
      cachedFile = null;
    }
  }
  return cachedFile?.[key];
}
