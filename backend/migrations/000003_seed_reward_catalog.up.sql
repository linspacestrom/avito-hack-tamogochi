-- Seed data for reward_definitions — a realistic starter catalog spanning all four
-- reward_type values, with a level spread from 1 to 20, per the mentor's guidance that
-- placeholders should look like plausible real cases, not empty/random stubs.
--
-- "value" shape by reward_type:
--   game_currency / game_energy: {"amount": <int>}
--   avito_promo:                 {"promo_code": "...", "feature": "...", ...}
--   avito_discount:              {"percent": <int>, "category": "..."}
--
-- ON CONFLICT (code) DO NOTHING makes this safe to re-run manually; golang-migrate itself
-- won't re-apply an already-recorded migration under normal operation.

INSERT INTO reward_definitions (code, title, description, required_level, validity_days, reward_type, value) VALUES
    ('welcome-coins', 'Приветственный бонус', 'Награда за создание питомца', 1, NULL,
        'game_currency', '{"amount": 50}'),

    ('energy-boost-small', 'Заряд бодрости', 'Полный заряд энергии для одной игры', 1, NULL,
        'game_energy', '{"amount": 20}'),

    ('promo-ad-boost', 'Бесплатное поднятие объявления', 'Промокод на бесплатное поднятие одного объявления в топ', 3, 7,
        'avito_promo', '{"promo_code": "PETBOOST3", "feature": "ad_boost", "uses": 1}'),

    ('discount-delivery-10', 'Скидка на доставку 10%', 'Скидка на следующую доставку через Авито Доставку', 5, 14,
        'avito_discount', '{"percent": 10, "category": "delivery"}'),

    ('discount-electronics-15', 'Скидка 15% в категории «Электроника»', 'Скидка на размещение или продвижение в категории Электроника', 7, 14,
        'avito_discount', '{"percent": 15, "category": "electronics"}'),

    ('coins-medium', 'Пакет монет', 'Средний пакет игровой валюты', 8, NULL,
        'game_currency', '{"amount": 150}'),

    ('promo-autoteka-report', 'Скидка на отчёт Автотеки', 'Скидка 20% на отчёт истории автомобиля в Автотеке', 10, 30,
        'avito_promo', '{"promo_code": "AUTOTEKA20", "feature": "autoteka_report", "discount_percent": 20}'),

    ('energy-full-refill', 'Полная перезарядка', 'Полное восстановление энергии', 12, NULL,
        'game_energy', '{"amount": 100}'),

    ('promo-featured-listing', 'Бесплатное VIP-размещение', 'Объявление в VIP-блоке на 3 дня', 15, 10,
        'avito_promo', '{"promo_code": "VIPWEEK15", "feature": "featured_listing", "duration_days": 3}'),

    ('discount-delivery-20', 'Скидка на доставку 20%', 'Повышенная скидка на доставку через Авито Доставку', 20, 14,
        'avito_discount', '{"percent": 20, "category": "delivery"}')
ON CONFLICT (code) DO NOTHING;
