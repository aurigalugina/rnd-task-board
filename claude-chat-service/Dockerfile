FROM node:20-bookworm-slim

# Host/network ini gak punya default route IPv6, tapi registry.npmjs.org (di
# belakang Cloudflare) balikin AAAA record juga -- Node 18+ defaultnya pakai
# urutan DNS 'verbatim' (ikut urutan dari resolver, bisa aja IPv6 duluan),
# beda dari Node <=16 yang defaultnya 'ipv4first'. Tanpa ini, npm ci/npm
# install nyoba connect ke alamat IPv6 yang gak ke-route dulu, baru fallback
# ke IPv4 -- gagal dengan ECONNRESET (bukan gagal cepat/jelas) di jaringan
# kayak ini. Force IPv4 duluan biar gak nyoba rute yang mati.
ENV NODE_OPTIONS=--dns-result-order=ipv4first

# node-pty butuh build tools (belum ada prebuilt binary buat linux di versi ini,
# lihat README "Catatan Docker & filesystem" — node-gyp compile dari source).
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 make g++ \
    && rm -rf /var/lib/apt/lists/*

# Jaringan build ini kadang putus sesaat (WSL2 + VPN mesh -- lihat
# decision-log-board-archive-20260812.md untuk temuan jaringan serupa di repo
# utama), keliatan dari koneksi yang ke-reset di tengah download besar
# (headers.tar.gz buat node-gyp, paket npm besar, dst). Retry-loop di sini
# (bukan ubah Docker daemon MTU -- itu bakal restart SEMUA container lain di
# mesin ini) supaya build tetap bisa sukses walau kena gangguan sesaat.
RUN for i in 1 2 3 4 5; do \
      npm install -g @anthropic-ai/claude-code && exit 0; \
      echo "npm install -g claude-code gagal (percobaan $i/5), retry 5s..."; \
      sleep 5; \
    done; exit 1

WORKDIR /app
COPY package.json package-lock.json ./
RUN for i in 1 2 3 4 5; do \
      npm ci && exit 0; \
      echo "npm ci gagal (percobaan $i/5), retry 5s..."; \
      sleep 5; \
    done; exit 1
COPY tsconfig.json ./
COPY src ./src
RUN npm run build

ENV NODE_ENV=production
EXPOSE 8090
CMD ["node", "dist/index.js"]
