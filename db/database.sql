SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

SET NAMES utf8mb4;

DROP TABLE IF EXISTS `posts`;
CREATE TABLE `posts` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `title` text COLLATE utf8mb4_unicode_520_ci NOT NULL,
  `image` longblob NOT NULL,
  `fileName` text COLLATE utf8mb4_unicode_520_ci NOT NULL,
  `createTime` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_520_ci;


DROP TABLE IF EXISTS `stats`;
CREATE TABLE `stats` (
  `view` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=latin1;