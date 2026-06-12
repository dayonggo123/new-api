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

import React from 'react';
import { Button, Switch } from '@douyinfe/semi-ui';

const PromptsActions = ({
  setEditingPrompt,
  setShowEdit,
  autoRefreshEnabled,
  setAutoRefreshEnabled,
  t,
}) => {
  // Add new prompt
  const handleAddPrompt = () => {
    setEditingPrompt({
      id: undefined,
    });
    setShowEdit(true);
  };

  return (
    <div className='flex flex-wrap items-center gap-2 w-full md:w-auto order-2 md:order-1'>
      <Button
        type='primary'
        className='flex-1 md:flex-initial'
        onClick={handleAddPrompt}
        size='small'
      >
        {t('添加提示词')}
      </Button>
      <div className='flex items-center gap-2 text-sm'>
        <Switch
          size='small'
          checked={autoRefreshEnabled}
          onChange={(checked) => setAutoRefreshEnabled(checked)}
          aria-label={t('自动刷新')}
        />
        <span className='text-gray-500'>{t('自动刷新')}</span>
      </div>
    </div>
  );
};

export default PromptsActions;
