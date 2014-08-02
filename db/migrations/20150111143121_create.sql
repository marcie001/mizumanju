
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE `user_display_settings` (
  `user_id` int(11) NOT NULL,
  `target_user_id` int(11) NOT NULL,
  `hide` tinyint(1) NOT NULL DEFAULT '0',
  `order_no` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`user_id`,`target_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_password_recovery` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` int(11) NOT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_status` (
  `user_id` int(11) NOT NULL,
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated` datetime NOT NULL,
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `auth_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `voice_chat_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `role` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password` char(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `delete_flag` tinyint(1) NOT NULL DEFAULT '0',
  `email` varchar(254) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `auth_id` (`auth_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
INSERT INTO `users` (`id`, `auth_id`, `name`, `voice_chat_id`, `role`, `password`, `created`, `delete_flag`, `email`) VALUES (1,'admin','管理 太郎','vcid001','admin','9c212d423f1615e504b8d3969da79600b2c0384508528da13af72d3f6ad10a3641d9603cb49a8a44b8e658054a06f9fd4370f5f3253d149ef0c321138d5597bf','2014-07-29 00:00:00',0,'admin@example.com');
INSERT INTO `users` (`id`, `auth_id`, `name`, `voice_chat_id`, `role`, `password`, `created`, `delete_flag`, `email`) VALUES (2,'user01','ユーザ 一郎','vcid002','editor','9c212d423f1615e504b8d3969da79600b2c0384508528da13af72d3f6ad10a3641d9603cb49a8a44b8e658054a06f9fd4370f5f3253d149ef0c321138d5597bf','2014-07-29 00:00:00',0,'user01@example.com');
INSERT INTO `users` (`id`, `auth_id`, `name`, `voice_chat_id`, `role`, `password`, `created`, `delete_flag`, `email`) VALUES (3,'user02','ユーザ ニ郎','vcid003','editor','9c212d423f1615e504b8d3969da79600b2c0384508528da13af72d3f6ad10a3641d9603cb49a8a44b8e658054a06f9fd4370f5f3253d149ef0c321138d5597bf','2014-07-29 00:00:00',0,'user02@example.com');

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE `user_display_settings`;
DROP TABLE `user_password_recovery`;
DROP TABLE `user_status`;
DROP TABLE `users`;
