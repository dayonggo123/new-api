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

import { useState, useEffect, useRef } from 'react';
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
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);

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

  // Batch translate state
  const [batchTranslating, setBatchTranslating] = useState(false);
  const [batchProgress, setBatchProgress] = useState({ current: 0, total: 0 });

  // Batch SEO translate state
  const [batchSEOTranslating, setBatchSEOTranslating] = useState(false);
  const [batchSEOProgress, setBatchSEOProgress] = useState({ current: 0, total: 0 });
  const seoPollIntervalRef = useRef(null);

  // Batch auto FAQ state
  const [batchAutoFAQTranslating, setBatchAutoFAQTranslating] = useState(false);
  const [batchAutoFAQProgress, setBatchAutoFAQProgress] = useState({ current: 0, total: 0 });
  const autoFaqPollIntervalRef = useRef(null);

  // Batch GEO blocks state
  const [batchGeoBlocksGenerating, setBatchGeoBlocksGenerating] = useState(false);
  const [batchGeoBlocksProgress, setBatchGeoBlocksProgress] = useState({ current: 0, total: 0 });
  const geoBlocksPollIntervalRef = useRef(null);

  // Row selection configuration
  const rowSelection = {
    selectedRowKeys,
    onChange: (keys, rows) => {
      setSelectedRowKeys(keys);
      setSelectedKeys(rows);
    },
  };

  // Batch translate handler
  const handleBatchTranslate = async () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择要翻译的提示词'));
      return;
    }
    setBatchTranslating(true);
    setBatchProgress({ current: 0, total: selectedRowKeys.length });
    const targetLangs = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'it', 'pt', 'ar'];

    for (let i = 0; i < selectedRowKeys.length; i++) {
      const id = selectedRowKeys[i];
      setBatchProgress({ current: i + 1, total: selectedRowKeys.length });
      try {
        const detailRes = await API.get(`/api/prompt/${id}`);
        if (!detailRes.data.success) continue;
        const item = detailRes.data.data;
        const items = [
          { key: 'title', text: item.title || '' },
          { key: 'content', text: item.content || '' },
          { key: 'content_en', text: item.content_en || '' },
          { key: 'description', text: item.description || '' },
        ].filter((it) => it.text !== '');
        if (items.length === 0) continue;

        const queueRes = await API.post('/api/translate/queue', {
          items,
          source_lang: 'zh',
          target_langs: targetLangs,
        });
        if (!queueRes.data.success) continue;
        const queueId = queueRes.data.data.queue_id;

        let results = null;
        let attempts = 0;
        const maxAttempts = 150;
        while (attempts < maxAttempts) {
          await new Promise((r) => setTimeout(r, 2000));
          const pollRes = await API.get(`/api/translate/queue/${queueId}`);
          const queue = pollRes.data.data;
          if (!queue) break;
          if (queue.status === 'done') {
            results = queue.results;
            break;
          } else if (queue.status === 'failed') break;
          attempts++;
        }
        if (!results) continue;

        const i18n = {};
        const titleI18n = {};
        Object.entries(results).forEach(([langCode, result]) => {
          if (result) {
            if (result.content) i18n[langCode] = result.content;
            if (result.title) titleI18n[langCode] = result.title;
          }
        });

        const payload = {
          title: item.title,
          content: item.content,
          content_en: item.content_en || '',
          description: item.description || '',
          cover_image_url: item.cover_image_url || '',
          author: item.author || '',
          model: item.model || '',
          category_id: item.category_id,
          media_type: item.media_type || 'image',
          variables: item.variables || '',
          tags: item.tags || '[]',
          sort_order: item.sort_order || 0,
          status: item.status,
          i18n: JSON.stringify(i18n),
          title_i18n: JSON.stringify(titleI18n),
          id: item.id,
        };
        await API.put('/api/prompt', payload);
      } catch (err) {
        console.error(`Batch translate failed for prompt ${id}:`, err);
      }
    }

    showSuccess(t('批量翻译完成'));
    setBatchTranslating(false);
    setSelectedRowKeys([]);
    setSelectedKeys([]);
    refresh();
  };

  // Batch SEO translate handler (async backend task)
  const handleBatchSEOTranslate = async () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择要翻译的提示词'));
      return;
    }
    if (selectedRowKeys.length > 50) {
      showError(t('单次最多选择 50 个提示词'));
      return;
    }
    // Clear any existing poll interval
    if (seoPollIntervalRef.current) {
      clearInterval(seoPollIntervalRef.current);
      seoPollIntervalRef.current = null;
    }

    setBatchSEOTranslating(true);
    setBatchSEOProgress({ current: 0, total: selectedRowKeys.length });
    const targetLangs = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'it', 'pt', 'ar'];

    try {
      const res = await API.post('/api/prompt/seo/batch-translate', {
        ids: selectedRowKeys,
        target_langs: targetLangs,
      });
      if (!res.data.success) {
        showError(res.data.message || t('启动批量 SEO 翻译失败'));
        setBatchSEOTranslating(false);
        return;
      }
      const taskId = res.data.data.task_id;
      showSuccess(res.data.data.message || t('已启动批量 SEO 翻译任务'));

      seoPollIntervalRef.current = setInterval(async () => {
        try {
          const statusRes = await API.get(`/api/prompt/seo/batch-translate/${taskId}`);
          const task = statusRes.data.data;
          if (!task) {
            clearInterval(seoPollIntervalRef.current);
            seoPollIntervalRef.current = null;
            setBatchSEOTranslating(false);
            return;
          }

          setBatchSEOProgress({
            current: task.completed || 0,
            total: task.total || selectedRowKeys.length,
          });

          if (task.status === 'completed' || task.status === 'failed') {
            clearInterval(seoPollIntervalRef.current);
            seoPollIntervalRef.current = null;
            setBatchSEOTranslating(false);
            setSelectedRowKeys([]);
            setSelectedKeys([]);

            const successCount = (task.completed || 0) - (task.failed || 0);
            const failCount = task.failed || 0;
            if (failCount > 0) {
              showSuccess(t('批量 SEO 翻译完成：') + successCount + t(' 成功，') + failCount + t(' 失败'));
            } else {
              showSuccess(t('批量 SEO 翻译全部完成'));
            }
            refresh();
          }
        } catch (err) {
          console.error('SEO batch translate poll error:', err);
        }
      }, 2000);
    } catch (err) {
      showError(err.message || t('启动批量 SEO 翻译失败'));
      setBatchSEOTranslating(false);
    }
  };

  // Batch auto FAQ handler (async backend task)
  const handleBatchAutoFAQ = async () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择要生成 FAQ 的提示词'));
      return;
    }
    if (selectedRowKeys.length > 20) {
      showError(t('单次最多选择 20 个提示词'));
      return;
    }
    // Clear any existing poll interval
    if (autoFaqPollIntervalRef.current) {
      clearInterval(autoFaqPollIntervalRef.current);
      autoFaqPollIntervalRef.current = null;
    }

    setBatchAutoFAQTranslating(true);
    setBatchAutoFAQProgress({ current: 0, total: selectedRowKeys.length });

    try {
      const res = await API.post('/api/admin/prompts/auto-faq/batch', {
        ids: selectedRowKeys,
      });
      if (!res.data.success) {
        showError(res.data.message || t('启动自动生成 FAQ 失败'));
        setBatchAutoFAQTranslating(false);
        return;
      }
      const taskId = res.data.data.task_id;
      showSuccess(res.data.data.message || t('已启动自动生成 FAQ 任务'));

      autoFaqPollIntervalRef.current = setInterval(async () => {
        try {
          const statusRes = await API.get(`/api/admin/auto-faq/batch/${taskId}`);
          const task = statusRes.data.data;
          if (!task) {
            clearInterval(autoFaqPollIntervalRef.current);
            autoFaqPollIntervalRef.current = null;
            setBatchAutoFAQTranslating(false);
            return;
          }

          setBatchAutoFAQProgress({
            current: task.completed || 0,
            total: task.total || selectedRowKeys.length,
          });

          if (task.status === 'completed' || task.status === 'failed') {
            clearInterval(autoFaqPollIntervalRef.current);
            autoFaqPollIntervalRef.current = null;
            setBatchAutoFAQTranslating(false);
            setSelectedRowKeys([]);
            setSelectedKeys([]);

            const successCount = (task.completed || 0) - (task.failed || 0);
            const failCount = task.failed || 0;
            if (failCount > 0) {
              showSuccess(t('自动生成 FAQ 完成：') + successCount + t(' 成功，') + failCount + t(' 失败'));
            } else {
              showSuccess(t('自动生成 FAQ 全部完成'));
            }
            refresh();
          }
        } catch (err) {
          console.error('Auto FAQ batch poll error:', err);
        }
      }, 2000);
    } catch (err) {
      showError(err.message || t('启动自动生成 FAQ 失败'));
      setBatchAutoFAQTranslating(false);
    }
  };

  const handleBatchGeoBlocks = async () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择要生成 GEO 结构的提示词'));
      return;
    }
    if (selectedRowKeys.length > 50) {
      showError(t('单次最多选择 50 个提示词'));
      return;
    }
    if (geoBlocksPollIntervalRef.current) {
      clearInterval(geoBlocksPollIntervalRef.current);
      geoBlocksPollIntervalRef.current = null;
    }

    setBatchGeoBlocksGenerating(true);
    setBatchGeoBlocksProgress({ current: 0, total: selectedRowKeys.length });

    try {
      const res = await API.post('/api/admin/prompts/geo-blocks/batch', {
        ids: selectedRowKeys,
      });
      if (!res.data.success) {
        showError(res.data.message || t('启动 GEO 结构生成失败'));
        setBatchGeoBlocksGenerating(false);
        return;
      }
      const taskId = res.data.data.task_id;
      showSuccess(res.data.data.message || t('已启动 GEO 结构生成任务'));

      geoBlocksPollIntervalRef.current = setInterval(async () => {
        try {
          const statusRes = await API.get(`/api/admin/geo-blocks/batch/${taskId}`);
          const task = statusRes.data.data;
          if (!task) {
            clearInterval(geoBlocksPollIntervalRef.current);
            geoBlocksPollIntervalRef.current = null;
            setBatchGeoBlocksGenerating(false);
            return;
          }

          setBatchGeoBlocksProgress({
            current: task.completed || 0,
            total: task.total || selectedRowKeys.length,
          });

          if (task.status === 'completed' || task.status === 'failed') {
            clearInterval(geoBlocksPollIntervalRef.current);
            geoBlocksPollIntervalRef.current = null;
            setBatchGeoBlocksGenerating(false);
            setSelectedRowKeys([]);
            setSelectedKeys([]);

            const successCount = (task.completed || 0) - (task.failed || 0);
            const failCount = task.failed || 0;
            if (failCount > 0) {
              showSuccess(t('GEO 结构生成完成：') + successCount + t(' 成功，') + failCount + t(' 失败'));
            } else {
              showSuccess(t('GEO 结构生成全部完成'));
            }
            refresh();
          }
        } catch (err) {
          console.error('GEO blocks batch poll error:', err);
        }
      }, 2000);
    } catch (err) {
      showError(err.message || t('启动 GEO 结构生成失败'));
      setBatchGeoBlocksGenerating(false);
    }
  };

  // Cleanup interval on unmount
  useEffect(() => {
    return () => {
      if (seoPollIntervalRef.current) {
        clearInterval(seoPollIntervalRef.current);
      }
      if (autoFaqPollIntervalRef.current) {
        clearInterval(autoFaqPollIntervalRef.current);
      }
      if (geoBlocksPollIntervalRef.current) {
        clearInterval(geoBlocksPollIntervalRef.current);
      }
    };
  }, []);

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
    selectedRowKeys,
    setSelectedRowKeys,
    batchTranslating,
    batchProgress,
    handleBatchTranslate,
    batchSEOTranslating,
    batchSEOProgress,
    handleBatchSEOTranslate,
    batchAutoFAQTranslating,
    batchAutoFAQProgress,
    handleBatchAutoFAQ,
    batchGeoBlocksGenerating,
    batchGeoBlocksProgress,
    handleBatchGeoBlocks,
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
