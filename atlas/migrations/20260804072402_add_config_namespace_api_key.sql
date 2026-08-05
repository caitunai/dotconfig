-- Modify "gb_config_namespaces" table
ALTER TABLE `gb_config_namespaces` ADD COLUMN `api_key` mediumtext NOT NULL AFTER `description`;
