-- Modify "gb_config_environments" table
ALTER TABLE `gb_config_environments` ADD COLUMN `draft_descriptions` mediumtext NOT NULL AFTER `draft_config`;
-- Modify "gb_config_releases" table
ALTER TABLE `gb_config_releases` ADD COLUMN `descriptions` mediumtext NOT NULL AFTER `config`;
