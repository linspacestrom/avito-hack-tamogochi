-- Seeds the 14-day daily reward calendar, reusing rows from reward_definitions rather than
-- inventing a parallel catalog. Value escalates toward the two milestone days (7 and 14) —
-- common pattern for daily-login calendars, gives a stronger reason not to miss those days.

INSERT INTO daily_reward_cycle (day_number, reward_definition_id)
SELECT day_number, (SELECT id FROM reward_definitions WHERE code = reward_code)
FROM (VALUES
    (1, 'welcome-coins'),
    (2, 'energy-boost-small'),
    (3, 'welcome-coins'),
    (4, 'energy-boost-small'),
    (5, 'welcome-coins'),
    (6, 'energy-boost-small'),
    (7, 'promo-ad-boost'),
    (8, 'welcome-coins'),
    (9, 'energy-boost-small'),
    (10, 'coins-medium'),
    (11, 'energy-full-refill'),
    (12, 'discount-electronics-15'),
    (13, 'energy-full-refill'),
    (14, 'discount-delivery-20')
) AS calendar(day_number, reward_code)
ON CONFLICT (day_number) DO NOTHING;
