# Contributing to Thask

## Development Setup

1. Fork and clone the repo
2. Copy `.env.example` to `.env`
3. Start PostgreSQL:
   ```bash
   docker compose -f docker-compose.dev.yml up -d
   ```
4. Install dependencies:
   ```bash
   npm install
   ```
5. Run database migrations:
   ```bash
   npm run db:push
   ```
6. Start the dev server:
   ```bash
   npm run dev
   ```

## Code Style

- TypeScript strict mode is enforced
- Run `npm run lint` and `npm run format` before committing
- Follow existing patterns in the codebase

## Pull Requests

- Create a feature branch from `main`
- Keep PRs focused on a single concern
- Ensure `npm run type-check` passes
- Include a clear description of the change

## Reporting Bugs

Use GitHub Issues with a clear description and steps to reproduce.
