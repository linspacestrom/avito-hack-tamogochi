-- Seed data for task_definitions — mirrors the three task types and their example lists from
-- ARCHITECTURE.md ("Задания": daily / general / avito_external), each task wired to an
-- existing reward_definitions row by code.
--
-- target_metric is the event type the future dispatcher will match against (see
-- ARCHITECTURE.md: "Отслеживание прогресса — через события... targetMetric совпадает с типом
-- события"). Values chosen here are placeholders for events that don't exist yet on main
-- (pet feeding/leveling/game-play events belong to the pet/minigame modules; avito_* events
-- are the external-action stubs the mentor confirmed are fine to mock for MVP).
--
-- reset_period: 'daily' for the three daily quests (reset by the future dispatcher once a
-- day); NULL for general/avito_external — those are one-time, not repeating.
--
-- ON CONFLICT (code) DO NOTHING makes this safe to re-run manually.

INSERT INTO task_definitions (code, type, title, description, target_metric, target_value, reward_definition_id, reset_period) VALUES
    -- Ежедневные
    ('daily-login', 'daily', 'Ежедневный вход', 'Зайти в приложение сегодня', 'daily_login', 1,
        (SELECT id FROM reward_definitions WHERE code = 'welcome-coins'), 'daily'),

    ('daily-feed-pet', 'daily', 'Покормить питомца', 'Покормить питомца хотя бы один раз сегодня', 'pet_feed', 1,
        (SELECT id FROM reward_definitions WHERE code = 'energy-boost-small'), 'daily'),

    ('daily-play-3-games', 'daily', 'Сыграть 3 раза', 'Сыграть в мини-игру 3 раза за сегодня', 'game_play', 3,
        (SELECT id FROM reward_definitions WHERE code = 'energy-boost-small'), 'daily'),

    -- Общие / долгосрочные
    ('reach-level-10', 'general', 'Достичь 10 уровня', 'Прокачать питомца до 10 уровня', 'pet_level', 10,
        (SELECT id FROM reward_definitions WHERE code = 'coins-medium'), NULL),

    ('earn-10000-xp', 'general', 'Заработать 10 000 опыта', 'Накопить 10 000 очков опыта питомца суммарно', 'xp_earned', 10000,
        (SELECT id FROM reward_definitions WHERE code = 'discount-electronics-15'), NULL),

    ('play-100-games', 'general', 'Сыграть 100 игр', 'Сыграть в мини-игру 100 раз суммарно', 'game_play', 100,
        (SELECT id FROM reward_definitions WHERE code = 'promo-featured-listing'), NULL),

    -- Вне игры (Авито) — заглушки с правдоподобными данными, реальные события подключаются после MVP
    ('avito-open-app', 'avito_external', 'Открыть приложение Авито', 'Открыть приложение Авито хотя бы один раз', 'avito_app_open', 1,
        (SELECT id FROM reward_definitions WHERE code = 'welcome-coins'), NULL),

    ('avito-view-listings', 'avito_external', 'Посмотреть объявления', 'Просмотреть 5 объявлений на Авито', 'avito_listing_view', 5,
        (SELECT id FROM reward_definitions WHERE code = 'energy-boost-small'), NULL),

    ('avito-post-listing', 'avito_external', 'Разместить объявление', 'Разместить одно объявление на Авито', 'avito_listing_post', 1,
        (SELECT id FROM reward_definitions WHERE code = 'promo-ad-boost'), NULL),

    ('avito-message-seller', 'avito_external', 'Написать продавцу', 'Написать сообщение продавцу по любому объявлению', 'avito_seller_message', 1,
        (SELECT id FROM reward_definitions WHERE code = 'energy-boost-small'), NULL),

    ('avito-arrange-delivery', 'avito_external', 'Оформить доставку', 'Оформить доставку через Авито Доставку', 'avito_delivery_arrange', 1,
        (SELECT id FROM reward_definitions WHERE code = 'discount-delivery-10'), NULL),

    ('avito-add-favorite', 'avito_external', 'Добавить в избранное', 'Добавить объявление в избранное', 'avito_favorite_add', 1,
        (SELECT id FROM reward_definitions WHERE code = 'welcome-coins'), NULL)
ON CONFLICT (code) DO NOTHING;
