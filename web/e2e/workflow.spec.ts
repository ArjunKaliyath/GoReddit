import { expect, test, type Page } from "@playwright/test";
async function register(page: Page, username: string) {
  await page.goto("/");
  await page.getByRole("button", { name: "Create an account" }).click();
  await page.getByLabel("Username", { exact: true }).fill(username);
  await page
    .getByLabel("Password", { exact: true })
    .fill("browser-test-password");
  await page.getByRole("button", { name: /^Create account/ }).click();
  await expect(
    page.getByRole("heading", { name: `Welcome back, ${username}.` }),
  ).toBeVisible();
}
test("two people can join, post, reply, vote, follow and exchange a message", async ({
  page,
  browser,
}) => {
  const suffix = Date.now().toString(36);
  const alice = `alice_${suffix}`,
    bob = `bobby_${suffix}`,
    community = `go_${suffix}`;
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await register(page, alice);
  await page.getByRole("button", { name: "Communities", exact: true }).click();
  await page.getByLabel("Community name").fill(community);
  await page
    .getByLabel("Description", { exact: true })
    .fill("A real community for Go developers");
  await page
    .getByRole("button", { name: "Create community", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: `r/${community}`, exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Home feed", exact: true }).click();
  await page.getByRole("button", { name: "+ Create post" }).click();
  await page
    .getByRole("combobox", { name: "Community", exact: true })
    .selectOption({ label: `r/${community}` });
  await page
    .getByLabel("Title", { exact: true })
    .fill("Hello from a real browser");
  await page
    .getByLabel("Your post")
    .fill("This is a persistent post, not a simulation.");
  await page.getByRole("button", { name: "Publish post" }).click();
  await expect(
    page.getByRole("heading", { name: "Hello from a real browser" }),
  ).toBeVisible();
  const context = await browser.newContext();
  const other = await context.newPage();
  await register(other, bob);
  await other.getByRole("button", { name: "Communities", exact: true }).click();
  const card = other.getByRole("article").filter({
    has: other.getByRole("heading", { name: `r/${community}`, exact: true }),
  });
  await card.getByRole("button", { name: "Join community" }).click();
  await expect(
    card.getByRole("button", { name: "Leave community" }),
  ).toBeVisible();
  await other.getByRole("button", { name: "Home feed", exact: true }).click();
  await expect(
    other.getByRole("heading", { name: "Hello from a real browser" }),
  ).toBeVisible();
  await other
    .getByRole("button", { name: "Upvote Hello from a real browser" })
    .click();
  await other.getByRole("button", { name: /Conversation/ }).click();
  await other.getByLabel("Your comment").fill("A real reply");
  await other.getByRole("button", { name: "Post comment" }).click();
  await expect(other.getByText("A real reply", { exact: true })).toBeVisible();
  await other.getByRole("button", { name: "People", exact: true }).click();
  const person = other.getByRole("article").filter({
    has: other.getByRole("heading", { name: `u/${alice}`, exact: true }),
  });
  await person.getByRole("button", { name: "Follow", exact: true }).click();
  await expect(
    person.getByRole("button", { name: "Unfollow", exact: true }),
  ).toBeVisible();
  await other.getByRole("button", { name: "Messages", exact: true }).click();
  await other.getByLabel("To", { exact: true }).fill(alice);
  await other
    .getByLabel("Message", { exact: true })
    .fill("Hello from another account");
  await other
    .getByRole("button", { name: "Send message", exact: true })
    .click();
  await expect(other.getByLabel("Message", { exact: true })).toHaveValue("");
  await page.getByRole("button", { name: "Messages", exact: true }).click();
  await expect(
    page.getByText("Hello from another account", { exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: /^Sign in/ })).toBeVisible();
  await page.getByLabel("Username", { exact: true }).fill(alice);
  await page
    .getByLabel("Password", { exact: true })
    .fill("browser-test-password");
  await page.getByRole("button", { name: /^Sign in/ }).click();
  await expect(
    page.getByRole("heading", { name: "Hello from a real browser" }),
  ).toBeVisible();
  await page.screenshot({ path: "test-results/desktop.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Communities", exact: true }),
  ).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    )
    .toBe(true);
  await page.screenshot({ path: "test-results/mobile.png", fullPage: true });
  expect(errors).toEqual([]);
  await context.close();
});
