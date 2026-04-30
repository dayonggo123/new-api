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
import CardPro from '../../common/ui/CardPro';
import PromptsTable from './PromptsTable';
import PromptsActions from './PromptsActions';
import PromptsFilters from './PromptsFilters';
import EditPromptModal from './modals/EditPromptModal';
import { usePromptsData } from '../../../hooks/prompts/usePromptsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const PromptsPage = () => {
  const promptsData = usePromptsData();
  const isMobile = useIsMobile();

  const {
    // Edit state
    showEdit,
    editingPrompt,
    closeEdit,
    refresh,
    categories,

    // Actions state
    setEditingPrompt,
    setShowEdit,

    // Filters state
    formInitValues,
    setFormApi,
    searchPrompts,
    loading,
    searching,

    // UI state
    compactMode,
    setCompactMode,

    // Translation
    t,
  } = promptsData;

  return (
    <>
      <EditPromptModal
        refresh={refresh}
        editingPrompt={editingPrompt}
        visiable={showEdit}
        handleClose={closeEdit}
        categories={categories}
      />

      <CardPro
        type='type1'
        descriptionArea={
          <div className='flex items-center gap-2'>
            <span className='text-sm font-medium'>{t('提示词管理')}</span>
          </div>
        }
        actionsArea={
          <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
            <PromptsActions
              setEditingPrompt={setEditingPrompt}
              setShowEdit={setShowEdit}
              t={t}
            />

            <div className='w-full md:w-full lg:w-auto order-1 md:order-2'>
              <PromptsFilters
                formInitValues={formInitValues}
                setFormApi={setFormApi}
                searchPrompts={searchPrompts}
                loading={loading}
                searching={searching}
                categories={categories}
                t={t}
              />
            </div>
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: promptsData.activePage,
          pageSize: promptsData.pageSize,
          total: promptsData.tokenCount,
          onPageChange: promptsData.handlePageChange,
          onPageSizeChange: promptsData.handlePageSizeChange,
          isMobile: isMobile,
          t: promptsData.t,
        })}
        t={promptsData.t}
      >
        <PromptsTable {...promptsData} />
      </CardPro>
    </>
  );
};

export default PromptsPage;
