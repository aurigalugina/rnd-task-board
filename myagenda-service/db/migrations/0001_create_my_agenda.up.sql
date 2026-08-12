CREATE TABLE `my_agenda` (
  `my_agenda_id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `judul_task` varchar(255) NOT NULL,
  `tgl_rencana` date NOT NULL,
  `uraian_task` text NOT NULL,
  `due_date` date NOT NULL,
  `target` double NOT NULL DEFAULT 0,
  `capaian` double NOT NULL DEFAULT 0,
  `is_percentage` tinyint(1) NOT NULL DEFAULT 0,
  `prosentase_capaian` int(11) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`my_agenda_id`),
  KEY `user_id` (`user_id`),
  KEY `tgl_rencana` (`tgl_rencana`),
  KEY `prosentase_capaian` (`prosentase_capaian`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
