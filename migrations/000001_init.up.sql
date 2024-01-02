CREATE TABLE `height` (
    `number` bigint UNSIGNED NOT NULL,
    `blockhash` char(66) NOT NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

CREATE TABLE `deposits` (
    `id` int UNSIGNED AUTO_INCREMENT,
    `txid` char(66) NOT NULL,
    `height` bigint UNSIGNED NOT NULL,
    `l1token` char(42) NOT NULL,
    `l2token` char(42) NOT NULL,
    `from` char(42) NOT NULL,
    `to` char(42) NOT NULL,
    `amount` decimal(64, 0) NOT NULL,
    `status` tinyint NOT NULL,
    `ctime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT pk_id PRIMARY KEY (`id`),
    INDEX idx_to (`to`),
    INDEX idx_txid(`txid`),
    INDEX idx_l2token(`l2token`),
    INDEX idx_height(`height`),
    INDEX idx_status (`status`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

CREATE TABLE `drips`(
    `pid` bigint UNSIGNED NOT NULL,
    `txid` char(66) NOT NULL,
    `from` char(42) NOT NULL,
    `to` char(42) NOT NULL,
    `amount` decimal(64, 20) NOT NULL,
    `rawtx` blob NOT NULL,
    `ctime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_pid PRIMARY KEY (`pid`),
    INDEX idx_to (`to`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;