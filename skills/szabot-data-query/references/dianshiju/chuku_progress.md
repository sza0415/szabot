```
USE szdw_dim;
CREATE TABLE `chuku_progress` (
    `db_name` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `table_name` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `imp_date` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
    `pv` bigint DEFAULT NULL,
    `imp_hour` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
    KEY `k` (`db_name`, `table_name`)
) ENGINE = InnoDB AUTO_INCREMENT = 134 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci
```
  【口径规则】
1. 查询时，必须用精确的 `table_name = '...'` 条件，**禁止**使用 `LIKE`、`IN` 等模糊/批量方式查询 `table_name`；所有合法的 `table_name` 值均已在 `schema/*.md` 中明确列出，直接使用即可。
2. 基于table_name取最大的imp_date，不用额外转换格式
3. 基于table_name取最大的imp_hour，不用额外转换格式