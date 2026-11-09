# Mobile workflow review

## Responsive paths reviewed

The app shell uses a mobile navigation drawer, a compact page title, and a constrained notification panel sized to the viewport. Core receipt list/detail and archive pages use responsive spacing and horizontal table scrolling where tabular fields cannot fit on a phone. The operational report uses the same responsive page shell.

## Manual verification scenarios (phone viewport, 360–390 px wide)

1. Sign in as requester; open the receipt list, create a receipt for the assigned department, and confirm the department cannot be changed.
2. Open a receipt detail from the list and from the reminder bell; confirm the action buttons and rejection reason field remain reachable without horizontal page scrolling.
3. Open the navigation drawer, visit PR archive and operational report, then close the drawer by choosing a route.
4. Open the reminder bell; check loading, empty, overdue, and API error states. Refresh the list and open a receipt.
5. Sign in as GA/approver; confirm status workflow controls remain usable and a rejection cannot be submitted without a reason.

Keyboard checks: tab through drawer, reminder, user menu, filters, and primary actions; verify visible focus and that Escape/backdrop can dismiss the drawer.

This checklist is the handoff for device-level UAT; completion should be confirmed against a running environment with representative accounts and data.
