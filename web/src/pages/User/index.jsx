/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState } from 'react';
import { Tabs, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import UsersTable from '../../components/table/users';
import TierManagement from '../TierManagement';
import TagManagement from '../TagManagement';

const { Text } = Typography;

const User = () => {
  const { t } = useTranslation();
  const [activeKey, setActiveKey] = useState('users');

  return (
    <div className='mt-[60px] px-2'>
      <div style={{ marginBottom: 16, padding: 12, background: '#f0f9ff', borderRadius: 8, borderLeft: '4px solid #0ea5e9' }}>
        <Text type='secondary' style={{ fontSize: 13, lineHeight: 1.6 }}>
          <strong>功能介绍：</strong>查看和管理注册用户，包括额度、权限、分组、状态<br/>
          <strong>如何操作：</strong>搜索用户，点击编辑修改额度/分组/状态，可查看用户的消费记录和 token 使用情况
        </Text>
      </div>
      <Tabs
        type='card'
        activeKey={activeKey}
        onChange={setActiveKey}
        style={{ marginBottom: 12 }}
      >
        <Tabs.TabPane itemKey='users' tab={t('用户列表')}>
          <UsersTable />
        </Tabs.TabPane>
        <Tabs.TabPane itemKey='tiers' tab={t('层级管理')}>
          <TierManagement />
        </Tabs.TabPane>
        <Tabs.TabPane itemKey='tags' tab={t('标签管理')}>
          <TagManagement />
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
};

export default User;
