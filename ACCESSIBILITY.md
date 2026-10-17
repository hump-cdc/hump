# Accessibility

Anyone who runs Hump should be able to use it, and anyone who wants to contribute should be able to. That includes people who use a screen reader, navigate by keyboard only, zoom the page, need high contrast or can't tell some colours apart.

This document covers the whole product: the web console, the CLI, logs, Prometheus metrics, and the repository's docs and issue and pull request templates. It also covers the contribution process. It doesn't cover the website, [hump.dev](https://hump.dev).

If something in Hump got in your way, go to [Reporting accessibility issues](#reporting-accessibility-issues).

Last reviewed: 2026-10-17

## Priorities

The target is WCAG 2.2 level AA. It's a goal, not a conformance claim: Hump hasn't been audited, and parts of it are known to fall short (see [Known limitations](#known-limitations)).

Work goes in this order:

1. Visual: enough contrast in both the light and the dark theme, status that is never shown by colour alone (operation badges, for example), and respect for the reduced-motion setting.
2. Text: plain language in docs, templates and error messages, and CLI and log output that never relies on colour alone to carry meaning.
3. Keyboard-only use and screen reader support.

## Contributor expectations

If you change the web console, `bun run check` in `web/webui` must pass. It runs Biome, and Biome's accessibility rules are part of its recommended rule set.

For any user-facing change, say in the Testing section of the pull request what you checked by hand:

- the change works with the keyboard only;
- it looks right in both the light and the dark theme;
- it still works at 200% zoom.

## Reporting accessibility issues

Open a [bug report](https://github.com/sergeyzwezdin/hump/issues/new?template=bug_report.yml) and select **Accessibility** under **Affected area**. For **Delivery impact**, choose **Not applicable**. You can leave **PostgreSQL version** empty.

If you have a question, or you aren't sure it's a bug, ask in [GitHub Discussions](https://github.com/sergeyzwezdin/hump/discussions).

A useful report says:

- what you were trying to do;
- the console page or the command;
- what happened;
- your browser and operating system;
- the assistive technology you use, if any, and relevant settings such as zoom level or theme.

Screenshots or recordings help but are optional. You don't need to tell us about any disability.

GitHub is the only channel; there is no email address. The maintainer watches issues and discussions, and accessibility issues get the `accessibility` label.

### How we respond

Hump has one maintainer, so accessibility work is best effort. You should get an acknowledgement within about 7 days. There are no guaranteed timelines after that.

Where a workaround exists, it's posted in the issue. Before the issue is closed, you'll be asked to confirm that the fix works for you.

## Ownership and maintenance

The maintainer owns accessibility: triaging reports, reviewing pull requests against the [contributor expectations](#contributor-expectations) and keeping this file current. If more maintainers join, the owner will be named here.

There is no review calendar. This file is reviewed when the web console changes in a way that affects accessibility, and when a limitation is found or fixed. The "Last reviewed" date at the top shows when that last happened.

## Supported environments

The web console has been tested in:

- desktop: the latest Chrome, Edge, Safari and Firefox;
- mobile: iOS and Android.

Keyboard-only use, 200% zoom and screen readers (VoiceOver, NVDA) haven't been evaluated yet ([#17](https://github.com/sergeyzwezdin/hump/issues/17)). Combinations not listed here may work, but nobody has checked.

The CLI, logs and metrics are plain text output.

## Known limitations

- Charts on the Statistics page have no text alternative. If you can't see the charts, you can't read the numbers in them. Workaround: the same counters are available as Prometheus metrics at `/metrics` on the telemetry port (`--telemetry-port`). [#15](https://github.com/sergeyzwezdin/hump/issues/15)
- Every console page has the same browser title, "hump", so tabs and screen reader page announcements don't tell the pages apart. [#16](https://github.com/sergeyzwezdin/hump/issues/16)
- Keyboard-only use, 200% zoom and screen readers haven't been evaluated. Barriers there are likely, but nobody has found and written them down yet. [#17](https://github.com/sergeyzwezdin/hump/issues/17)

## Feedback and improvements

- To suggest a change to how the project handles accessibility, use the [feature request form](https://github.com/sergeyzwezdin/hump/issues/new?template=feature_request.yml).
- To fix something in this document, open a pull request that edits `ACCESSIBILITY.md`.
- If you hit a barrier now, follow [Reporting accessibility issues](#reporting-accessibility-issues).
