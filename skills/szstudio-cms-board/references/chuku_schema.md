```
USE szdw_dim;
CREATE TABLE `chuku_progress` (
    `id` bigint NOT NULL AUTO_INCREMENT,
    `db_name` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `table_name` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `imp_date` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `ETL_STAMP` varchar(200) COLLATE utf8mb4_general_ci DEFAULT NULL,
    `pv` bigint DEFAULT NULL,
    `imp_hour` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `k` (`db_name`, `table_name`)
) ENGINE = InnoDB AUTO_INCREMENT = 134 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci
```

## 【口径规则】
1. 查询时，需要指定table_name
2. 基于table_name取最大的imp_date，不用额外转换格式
3. 基于table_name取最大的imp_hour，不用额外转换格式
