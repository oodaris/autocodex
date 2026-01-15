import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const routes = ['/', '/memory', '/terminal', '/hub']

test.describe('a11y smoke', () => {
  for (const route of routes) {
    test(`page ${route} has no critical a11y violations`, async ({ page }) => {
      await page.goto(route)
      const results = await new AxeBuilder({ page }).analyze()
      const serious = results.violations.filter((violation) =>
        ['critical', 'serious'].includes(violation.impact ?? 'minor'),
      )
      expect(serious).toEqual([])
    })
  }
})
