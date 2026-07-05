/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_IM_API_BASE: string
  readonly VITE_IM_WS_BASE: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
