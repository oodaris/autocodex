import { expect, test } from '@playwright/test'

test.describe('autocodex control deck', () => {
  test('dashboard loads', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: /keep the loop moving/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /run summary/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /recent runs/i })).toBeVisible()
  })

  test('memory page loads', async ({ page }) => {
    await page.goto('/memory')
    await expect(page.getByRole('heading', { name: /memory docs/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /documents/i })).toBeVisible()
  })

  test('terminal page loads', async ({ page }) => {
    await page.goto('/terminal')
    await expect(page.getByRole('heading', { name: /terminal sessions/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /start session/i })).toBeVisible()
  })

  test('hub page loads', async ({ page }) => {
    await page.goto('/hub')
    await expect(page.getByRole('heading', { name: /hub workspaces/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /configured repos/i })).toBeVisible()
  })
})
