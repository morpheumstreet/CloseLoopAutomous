/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ARMS_URL?: string;
  readonly VITE_ARMS_TOKEN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
