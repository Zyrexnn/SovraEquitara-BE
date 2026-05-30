-- SovraEquitara Database Backup
-- Generated at: 2026-05-29T21:35:29+07:00

SET session_replication_role = 'replica';

-- TRUNCATE profiles;
INSERT INTO profiles (id, email, password_hash, full_name, phone, avatar_url, points, role, created_at, updated_at) VALUES
('6ee24333-eade-430b-a49f-242c7c69b77c', 'ikhsan@admin.com', '$2a$10$gnEwx8Vd5jbg.lP1AeR5TuJcjud1.lMuaNnUHDXkf2mj0PYL.PhCe', 'Admin Ikhsan', '', '/uploads/avatar-6ee24333-eade-430b-a49f-242c7c69b77c-1779013160-be1b82d6c21fc3c8f62aa1ed57deab72.jpg', 0, 'admin', '2026-05-06 09:15:35.548114+00', '2026-05-17 10:19:20.560231+00'),
('ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'iksannovriansyah5@gmail.com', '$2a$10$f00DXeYU3vYWcCImDAFVSOxy94Fy8tGU8fCRX8F9d6B5aKyJd6CgW', 'iksan', '424242', '/uploads/avatar-ba39e85d-0439-4fe3-8835-6a47f5bab72d-1779013069-f9e5aaded1ee0a9ad2402ae3f428b2b9.jpg', 120, 'USER', '2026-05-06 12:46:30.091285+00', '2026-05-26 14:22:15.623710+00'),
('90584293-df7f-44f1-bb9b-bf2fbd026977', 'indo@admin.com', '$2a$10$EUBZYdycKfljfT/Vg6rHRugab3JbUfex.z3kArs4J1rnCIuE1ipzy', 'indonesia', '', NULL, 0, 'admin', '2026-05-29 10:30:22.490893+00', '2026-05-29 10:30:22.490893+00'),
('a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'super@admin.com', '$2a$10$1nGUZiHy4smydg/V5iZWdeYpMwnThdd8.3dR12J7O34uWJJv.hmHm', 'Super Admin', '', '/uploads/avatar-a23dfcb2-b3eb-4abf-80aa-1a91ed567d67-1779075564-WIN_20260421_15_40_38_Pro.jpg', 0, 'super_admin', '2026-05-13 02:13:38.420219+00', '2026-05-29 10:44:41.492910+00');

-- TRUNCATE categories;
INSERT INTO categories (id, name, slug) VALUES
(1, 'Infrastruktur', 'infrastruktur'),
(2, 'Lingkungan', 'lingkungan'),
(3, 'Fasilitas Umum', 'fasilitas-umum'),
(4, 'Keamanan', 'keamanan');

-- TRUNCATE reports;
INSERT INTO reports (id, profile_id, category_id, image_urls, description, phone_number, latitude, longitude, location_detail, vote_count, comment_count, status, created_at, updated_at) VALUES
('166931a1-3657-4926-8bf8-2d6ea3335c06', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 2, ARRAY['/uploads/1778471335-WIN_20260114_11_48_11_Pro.jpg']::TEXT[], 'Saya akan lawan', NULL, -6.384264, 106.870027, 'SMK TARUNA BHAKTI', 1, 0, 'VALID', '2026-05-11 03:48:55.645755+00', '2026-05-22 12:16:54.401246+00'),
('32463740-6981-48e6-9354-2c1cd5ee98da', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 4, ARRAY['/uploads/1778139964-supabase-schema-default.png']::TEXT[], 'rrwwrwrw', NULL, -6.384250, 106.869930, 'eqioequ9quq', 0, 0, 'RESOLVED', '2026-05-07 07:46:04.448935+00', '2026-05-22 12:16:54.401246+00'),
('df71f78a-4da0-4ab6-95a6-d440ae6393f2', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 3, ARRAY['/uploads/1778633020-MEXC_POSTER (18).png']::TEXT[], 'aku  raja kripto', NULL, -6.384211, 106.869947, 'SMK TARUNA BHAKTI', 1, 0, 'RESOLVED', '2026-05-13 00:43:40.168482+00', '2026-05-22 12:16:54.401246+00'),
('75846efc-52b1-4bb8-ab80-cc8ef9101539', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 2, ARRAY['/uploads/1778345065-photo_2026-05-05_14-24-45.jpg']::TEXT[], 'hidup jokowi', NULL, -6.384264, 106.870027, 'adalah', 1, 0, 'VALID', '2026-05-09 16:44:25.056898+00', '2026-05-22 12:22:48.224816+00'),
('9da4cb9d-ef40-4358-8328-0a001496f8c7', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 2, ARRAY['/uploads/1779442268-WIN_20260512_16_34_18_Pro.jpg']::TEXT[], 'afsvklnms sklwas', NULL, -6.175402, 106.827169, 'dadadqd1e1fw', 0, 0, 'WAITING_APPROVAL', '2026-05-22 09:31:08.152302+00', '2026-05-23 09:28:30.540989+00'),
('b21326fa-a273-4799-9a40-50522bdcda08', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 3, ARRAY['/uploads/1778073053-dashboard.png']::TEXT[], 'CACLACAOCADAV', NULL, -6.423557, 106.862871, 'DALDADKAD', 1, 0, 'VALID', '2026-05-06 13:10:53.268352+00', '2026-05-26 14:22:44.838482+00'),
('6d2db939-3e26-4bce-b7ae-6af67badfa99', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 1, ARRAY['/uploads/1779238936-WIN_20260512_16_34_00_Pro.jpg']::TEXT[], 'adalah', NULL, -6.384264, 106.870027, 'SMK TARUNA BHAKTI', 0, 0, 'PENDING', '2026-05-20 01:02:16.994906+00', '2026-05-29 10:44:41.492910+00');

-- TRUNCATE comments;
INSERT INTO comments (id, report_id, user_id, content, created_at) VALUES
('62c50666-4ddd-4cf5-bf7c-af08c14f844d', 'b21326fa-a273-4799-9a40-50522bdcda08', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'lah', '2026-05-06 13:15:13.615076+00'),
('453b7d7b-f4ea-4e9c-8739-ad5afbf5c5a3', 'b21326fa-a273-4799-9a40-50522bdcda08', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'loh', '2026-05-07 07:22:32.515001+00'),
('400eaec0-5426-4cc6-a116-9300085f4a2c', '75846efc-52b1-4bb8-ab80-cc8ef9101539', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'test', '2026-05-12 02:14:10.772344+00'),
('03fd7c22-2c07-4968-a705-2b0566678e1e', '166931a1-3657-4926-8bf8-2d6ea3335c06', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'JAWA', '2026-05-18 03:30:16.979118+00'),
('baeb732e-b330-4c0e-82b8-534084d506c5', 'df71f78a-4da0-4ab6-95a6-d440ae6393f2', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'test', '2026-05-22 07:49:50.626641+00');

-- TRUNCATE votes;
INSERT INTO votes (user_id, report_id, vote_type) VALUES
('ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'b21326fa-a273-4799-9a40-50522bdcda08', 1),
('ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'df71f78a-4da0-4ab6-95a6-d440ae6393f2', 1),
('ba39e85d-0439-4fe3-8835-6a47f5bab72d', '166931a1-3657-4926-8bf8-2d6ea3335c06', 1),
('ba39e85d-0439-4fe3-8835-6a47f5bab72d', '75846efc-52b1-4bb8-ab80-cc8ef9101539', 1);

-- TRUNCATE saved_reports;
INSERT INTO saved_reports (admin_id, report_id, created_at) VALUES
('6ee24333-eade-430b-a49f-242c7c69b77c', '166931a1-3657-4926-8bf8-2d6ea3335c06', '2026-05-20 04:29:17.787279+00'),
('a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'df71f78a-4da0-4ab6-95a6-d440ae6393f2', '2026-05-29 14:03:12.898708+00');

-- TRUNCATE conversations;
INSERT INTO conversations (id, participant_id, last_message, last_message_at, unread_count, created_at, updated_at) VALUES
('2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'dgdgdgd\', '2026-05-17 10:18:07.437980+00', 0, '2026-05-13 10:51:32.057742+00', '2026-05-29 01:05:33.309460+00'),
('ae732faf-19a6-4932-ab0e-f2b649255480', '6ee24333-eade-430b-a49f-242c7c69b77c', 'adad', '2026-05-26 14:33:08.378643+00', 0, '2026-05-14 06:47:26.318618+00', '2026-05-29 10:28:54.332768+00');

-- TRUNCATE messages;
INSERT INTO messages (id, conversation_id, sender_id, content, is_read, created_at) VALUES
('8cdb24f8-6711-4f18-a16a-d3f0940391cb', '2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'halo', true, '2026-05-13 10:51:37.981569+00'),
('db68a543-9899-44a3-ac01-f6d446f02ac5', '2af442ca-df46-4dae-be85-ac227d273940', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'd', true, '2026-05-13 10:52:56.822529+00'),
('9951d8d8-fac3-4d8c-8c52-641c8b523296', '2af442ca-df46-4dae-be85-ac227d273940', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'ada apa?', true, '2026-05-13 11:24:54.616182+00'),
('43e08ad8-69e7-46b0-b7d6-d23b99e2727f', 'ae732faf-19a6-4932-ab0e-f2b649255480', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'tes', true, '2026-05-14 06:47:48.785144+00'),
('b27cdac9-123d-41e9-8ca1-59af5a514f4d', '2af442ca-df46-4dae-be85-ac227d273940', '6ee24333-eade-430b-a49f-242c7c69b77c', 'w', true, '2026-05-14 12:11:17.356749+00'),
('db6ee854-5d6c-4356-85e8-5d3abcc3b8a0', '2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'dadada', true, '2026-05-17 08:13:41.994646+00'),
('a2ac9bc4-1ca4-42b9-ace6-7691e931a135', '2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'adalah', true, '2026-05-17 08:14:17.931249+00'),
('91dda2cd-65d4-4e7c-af80-4d980d6ca316', '2af442ca-df46-4dae-be85-ac227d273940', '6ee24333-eade-430b-a49f-242c7c69b77c', 'dada', true, '2026-05-17 08:38:06.160217+00'),
('aa741d4b-0c92-4cf0-b557-9ffd6a8b02d4', 'ae732faf-19a6-4932-ab0e-f2b649255480', '6ee24333-eade-430b-a49f-242c7c69b77c', 'daa', true, '2026-05-17 08:38:10.691471+00'),
('cc4c3426-a428-45ff-8daa-16f32bfc4a39', 'ae732faf-19a6-4932-ab0e-f2b649255480', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'dadad', true, '2026-05-17 08:39:18.811498+00'),
('89d15d64-5cff-418c-b477-bfb45ea47912', '2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'gdgdgdgd', true, '2026-05-17 08:43:23.163985+00'),
('bd6a2f01-9cd3-425e-9f3b-4ebbac11f3bb', '2af442ca-df46-4dae-be85-ac227d273940', '6ee24333-eade-430b-a49f-242c7c69b77c', 'oek', true, '2026-05-17 08:44:09.404105+00'),
('bb2af1e2-d18a-4bb6-8234-0f2a35788603', '2af442ca-df46-4dae-be85-ac227d273940', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'Alafa fnqaikldqopdqkbfh', true, '2026-05-17 08:44:53.569634+00'),
('e2aca33f-c0d7-40d3-a023-ba99bbd419d3', '2af442ca-df46-4dae-be85-ac227d273940', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', 'dgdgdgd\', true, '2026-05-17 10:18:07.419022+00'),
('e5bddb41-7011-47de-8f33-cb4c701993db', 'ae732faf-19a6-4932-ab0e-f2b649255480', '6ee24333-eade-430b-a49f-242c7c69b77c', ', lm l', true, '2026-05-19 07:18:55.384808+00'),
('58a3aa5c-9b65-4834-83ba-3076cfbd766e', 'ae732faf-19a6-4932-ab0e-f2b649255480', '6ee24333-eade-430b-a49f-242c7c69b77c', 'woi', true, '2026-05-23 14:15:31.112572+00'),
('c40806f7-e6c7-4c46-a62d-be24b44607d3', 'ae732faf-19a6-4932-ab0e-f2b649255480', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'ape', true, '2026-05-23 14:15:59.346533+00'),
('20f518d4-a3c8-4cd1-acdf-1589f812bd7c', 'ae732faf-19a6-4932-ab0e-f2b649255480', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'wou', true, '2026-05-25 03:32:42.253262+00'),
('dc05e7c5-4b14-456d-a883-a2d268cbb4f6', 'ae732faf-19a6-4932-ab0e-f2b649255480', 'a23dfcb2-b3eb-4abf-80aa-1a91ed567d67', 'adad', true, '2026-05-26 14:33:08.369683+00');

-- TRUNCATE notifications;
INSERT INTO notifications (id, title, message, type, target_role, target_user_id, action_url, created_by, created_at) VALUES
('5c884a04-f7d2-46bb-befc-e34341f17ead', 'Laporan Diverifikasi', 'Laporan Anda telah diverifikasi oleh petugas dan berstatus VALID.', 'INFO', 'SPECIFIC_USER', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', '/history?open=9da4cb9d-ef40-4358-8328-0a001496f8c7', NULL, '2026-05-23 09:28:26.724987+00'),
('4a7c2e7f-1e9a-45a3-ada3-7c84166876bd', 'Laporan Ditangani', 'Laporan Anda telah ditangani dan menunggu konfirmasi/persetujuan Anda.', 'INFO', 'SPECIFIC_USER', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', '/history?open=9da4cb9d-ef40-4358-8328-0a001496f8c7', NULL, '2026-05-23 09:28:30.545261+00'),
('b180502c-73ff-4ed0-a26a-42588aa0456f', 'Laporan Diverifikasi', 'Laporan Anda telah diverifikasi oleh petugas dan berstatus VALID.', 'INFO', 'SPECIFIC_USER', 'ba39e85d-0439-4fe3-8835-6a47f5bab72d', '/history?open=b21326fa-a273-4799-9a40-50522bdcda08', NULL, '2026-05-26 14:22:44.842211+00');

SET session_replication_role = 'origin';
