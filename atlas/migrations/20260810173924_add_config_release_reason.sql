-- Modify "dc_config_releases" table
ALTER TABLE `dc_config_releases` ADD COLUMN `reason` varchar(500) NOT NULL DEFAULT "" AFTER `batch_id`;
