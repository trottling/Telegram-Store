import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    server: {
        port: 3000,
    },
    build: {
        // Every page is its own lazy-loaded chunk (see App.tsx) — the main
        // bundle is ~120kB. StatsPage is the one legitimately heavy
        // exception (pulls in @ant-design/plots, a full charting lib on G2)
        // and it's isolated to its own chunk that only loads when an admin
        // actually opens Statistics, so warning about its size specifically
        // is just noise. Limit set just above its current ~1.46MB so a
        // regression elsewhere still warns.
        chunkSizeWarningLimit: 1600,
    },
})
