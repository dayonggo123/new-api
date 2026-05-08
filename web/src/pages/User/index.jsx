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
import { Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import UsersTable from '../../components/table/users';
import TierManagement from '../TierManagement';
import TagManagement from '../TagManagement';

const User = () => {
  const { t } = useTranslation();
  const [activeKey, setActiveKey] = useState('users');

  return (
    <div className='mt-[60px] px-2'>
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
