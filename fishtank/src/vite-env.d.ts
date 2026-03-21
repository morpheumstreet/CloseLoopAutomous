/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ARMS_URL?: string;
  readonly VITE_ARMS_TOKEN?: string;
  readonly VITE_ARMS_BASIC_USER?: string;
  readonly VITE_ARMS_BASIC_PASSWORD?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
