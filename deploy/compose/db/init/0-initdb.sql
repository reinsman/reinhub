CREATE DATABASE IF NOT EXISTS `example`
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS 'app.example'@'%' IDENTIFIED BY 'example-dev-password';
GRANT ALL PRIVILEGES ON `example`.* TO 'app.example'@'%';

FLUSH PRIVILEGES;
