/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Set by Vite in dev; Bun production builds may omit `import.meta.env` — use `isDevBuild()` from armsEnv. */
  readonly DEV?: boolean;
  readonly VITE_ARMS_URL?: string;
  readonly VITE_ARMS_TOKEN?: string;
  readonly VITE_ARMS_BASIC_USER?: string;
  readonly VITE_ARMS_BASIC_PASSWORD?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
