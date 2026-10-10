FROM node:22-alpine

WORKDIR /app

# Install pnpm (pin v10 to avoid strict build-script policy in v11)
RUN corepack enable pnpm && corepack prepare pnpm@10 --activate

# Copy package management files
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy the rest of the application
COPY . .

# Build the Next.js application
# We provide dummy values for build-time env vars if Next requires them
ENV NEXT_PUBLIC_SUPABASE_URL=http://localhost:54321
ENV NEXT_PUBLIC_SUPABASE_ANON_KEY=dummy
ENV CI=true
RUN pnpm build

# Set production environment
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

EXPOSE 3000

# Start the application
CMD ["pnpm", "start"]
