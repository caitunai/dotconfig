-- Create "gb_config_namespaces" table
CREATE TABLE `gb_config_namespaces` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `slug` varchar(64) NOT NULL,
  `description` varchar(500) NOT NULL DEFAULT "",
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_gb_config_namespaces_slug` (`slug`)
) CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
-- Give existing environments a namespace before enforcing the foreign key.
INSERT INTO `gb_config_namespaces` (`name`, `slug`, `description`) VALUES ('默认命名空间', 'default', '用于承接升级前已有的配置环境');
-- Modify "gb_config_environments" table
ALTER TABLE `gb_config_environments` ADD COLUMN `namespace_id` bigint NOT NULL DEFAULT 1 AFTER `name`, DROP INDEX `idx_gb_config_environments_slug`, ADD UNIQUE INDEX `idx_config_environment_namespace_slug` (`namespace_id`, `slug`), ADD CONSTRAINT `fk_gb_config_environments_namespace` FOREIGN KEY (`namespace_id`) REFERENCES `gb_config_namespaces` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE `gb_config_environments` ALTER COLUMN `namespace_id` DROP DEFAULT;
