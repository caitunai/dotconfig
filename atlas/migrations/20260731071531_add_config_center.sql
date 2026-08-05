-- Create "gb_config_environments" table
CREATE TABLE `gb_config_environments` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `slug` varchar(64) NOT NULL,
  `description` varchar(500) NOT NULL DEFAULT "",
  `parent_id` bigint NOT NULL DEFAULT 0,
  `draft_config` mediumtext NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_gb_config_environments_parent_id` (`parent_id`),
  UNIQUE INDEX `idx_gb_config_environments_slug` (`slug`)
) CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
-- Create "gb_config_releases" table
CREATE TABLE `gb_config_releases` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `environment_id` bigint NOT NULL,
  `batch_id` varchar(32) NOT NULL,
  `version` bigint NOT NULL,
  `config` mediumtext NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_config_release_version` (`environment_id`, `version`),
  INDEX `idx_gb_config_releases_batch_id` (`batch_id`),
  CONSTRAINT `fk_gb_config_releases_environment` FOREIGN KEY (`environment_id`) REFERENCES `gb_config_environments` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
) CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
