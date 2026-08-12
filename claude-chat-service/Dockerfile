FROM node:20-bookworm-slim

# node-pty butuh build tools (belum ada prebuilt binary buat linux di versi ini,
# lihat README "Catatan Docker & filesystem" — node-gyp compile dari source).
RUN apt-get update && apt-get install -y --no-install-recursive \
    python3 make g++ \
    && rm -rf /var/lib/apt/lists/*

# Claude Code CLI global — dipakai routes/auth.ts (setup-token, auth status),
# TERPISAH dari @anthropic-ai/claude-agent-sdk (yang itu bundle binary sendiri
# buat query() chat session, tidak butuh CLI global ini).
RUN npm install -g @anthropic-ai/claude-code

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json ./
COPY src ./src
RUN npm run build

ENV NODE_ENV=production
EXPOSE 8090
CMD ["node", "dist/index.js"]
