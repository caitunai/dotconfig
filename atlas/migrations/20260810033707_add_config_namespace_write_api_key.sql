-- Modify "dc_config_namespaces" table
ALTER TABLE `dc_config_namespaces` ADD COLUMN `write_api_key` mediumtext NOT NULL AFTER `api_key`;
