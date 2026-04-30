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

import { useState, useEffect } from 'react';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTranslation } from 'react-i18next';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const usePromptsData = () => {
  const { t } = useTranslation();

  // Basic state
  const [prompts, setPrompts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [tokenCount, setTokenCount] = useState(0);
  const [selectedKeys, setSelectedKeys] = useState([]);

  // Categories state
  const [categories, setCategories] = useState([]);

  // Edit state
  const [editingPrompt, setEditingPrompt] = useState({
    id: undefined,
  });
  const [showEdit, setShowEdit] = useState(false);

  // Form API
  const [formApi, setFormApi] = useState(null);

  // UI state
  const [compactMode, setCompactMode] = useTableCompactMode('prompts');

  // Form state
  const formInitValues = {
    searchKeyword: '',
    categoryId: '',
  };

  // Get form values
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      categoryId: formValues.categoryId || '',
    };
  };

  // Load categories
  const loadCategories = async () => {
    try {
      const res = await API.get('/api/prompt-category/all');
      const { success, message, data } = res.data;
      if (success) {
        setCategories(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  // Load prompt list
  const loadPrompts = async (page = 1, pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/prompt/?p=${page}&page_size=${pageSize}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page <= 0 ? 1 : data.page);
        setTokenCount(data.total);
        setPrompts(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  // Search prompts
  const searchPrompts = async () => {
    const { searchKeyword, categoryId } = getFormValues();

    if (searchKeyword === '' && categoryId === '') {
      await loadPrompts(1, pageSize);
      return;
    }

    setSearching(true);
    try {
      let url = `/api/prompt/?p=1&page_size=${pageSize}`;
      if (searchKeyword) {
        url += `&keyword=${encodeURIComponent(searchKeyword)}`;
      }
      if (categoryId) {
        url += `&category_id=${categoryId}`;
      }
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page || 1);
        setTokenCount(data.total);
        setPrompts(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setSearching(false);
  };

  // Delete prompt
  const deletePrompt = async (id) => {
    setLoading(true);
    try {
      const res = await API.delete(`/api/prompt/${id}/`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('操作成功完成！'));
        await refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  // Refresh data
  const refresh = async (page = activePage) => {
    const { searchKeyword, categoryId } = getFormValues();
    if (searchKeyword === '' && categoryId === '') {
      await loadPrompts(page, pageSize);
    } else {
      await searchPrompts();
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setActivePage(page);
    const { searchKeyword, categoryId } = getFormValues();
    if (searchKeyword === '' && categoryId === '') {
      loadPrompts(page, pageSize);
    } else {
      searchPrompts();
    }
  };

  // Handle page size change
  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setActivePage(1);
    const { searchKeyword, categoryId } = getFormValues();
    if (searchKeyword === '' && categoryId === '') {
      loadPrompts(1, size);
    } else {
      searchPrompts();
    }
  };

  // Row selection configuration
  const rowSelection = {
    onSelect: (record, selected) => {},
    onSelectAll: (selected, selectedRows) => {},
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows);
    },
  };

  // Close edit modal
  const closeEdit = () => {
    setShowEdit(false);
    setTimeout(() => {
      setEditingPrompt({
        id: undefined,
      });
    }, 500);
  };

  // Initialize data loading
  useEffect(() => {
    loadPrompts(1, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    loadCategories();
  }, [pageSize]);

  return {
    // Data state
    prompts,
    loading,
    searching,
    activePage,
    pageSize,
    tokenCount,
    selectedKeys,
    categories,

    // Edit state
    editingPrompt,
    showEdit,

    // Form state
    formApi,
    formInitValues,

    // UI state
    compactMode,
    setCompactMode,

    // Data operations
    loadPrompts,
    searchPrompts,
    deletePrompt,
    refresh,
    loadCategories,

    // State updates
    setActivePage,
    setPageSize,
    setSelectedKeys,
    setEditingPrompt,
    setShowEdit,
    setFormApi,
    setLoading,

    // Event handlers
    handlePageChange,
    handlePageSizeChange,
    rowSelection,
    closeEdit,
    getFormValues,

    // Translation function
    t,
  };
};
