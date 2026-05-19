import { defineConfig } from 'drizzle-kit'

export default defineConfig({
  schema: './src/main/data/schema.ts',
  out: './drizzle',
  dialect: 'sqlite',
  dbCredentials: {
    url: './.tmp/migration-temp.db',
  },
})
