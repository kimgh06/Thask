FROM node:22-alpine

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=7241

COPY package*.json ./
RUN npm ci --omit=dev
COPY src ./src

USER node
EXPOSE 7241
CMD ["node", "src/server.js"]
