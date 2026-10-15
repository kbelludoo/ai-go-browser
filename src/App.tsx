import React, { useState, useEffect } from 'react';
import { browserEngineInstance } from './engine/browserEngine';
import { SAMPLE_PAGES, SamplePageDefinition } from './engine/samplePages';
import { Page, Action, Resource, Result, ElementMatch, Command } from './types/browser';
import { Header } from './components/Header';
import { CommandConsole } from './components/CommandConsole';
import { PageViewer } from './components/PageViewer';
import { ActionsTable } from './components/ActionsTable';
import { DOMInspector } from './components/DOMInspector';
import { MarkdownViewer } from './components/MarkdownViewer';
import { TokenBenchmark } from './components/TokenBenchmark';
import { ArenaServicePanel } from './components/ArenaServicePanel';
import { RealSitesTestRunner } from './components/RealSitesTestRunner';

export function App() {
  const [activeTab, setActiveTab] = useState<string>('browser');
  const [activeLayer, setActiveLayer] = useState<1 | 2 | 3>(1);
  const [currentPage, setCurrentPage] = useState<Page | null>(null);
  const [currentHTML, setCurrentHTML] = useState<string>('');
  const [actions, setActions] = useState<Action[]>([]);
  const [selectedAction, setSelectedAction] = useState<Action | null>(null);
  const [lastResource, setLastResource] = useState<Resource | null>(null);
  const [lastResult, setLastResult] = useState<Result | null>(null);
  const [lastMatches, setLastMatches] = useState<ElementMatch[] | null>(null);
  const [activeSampleId, setActiveSampleId] = useState<string>('demo-local');

  // Synchronize initial state
  useEffect(() => {
    const init = async () => {
      const snapResult = await browserEngineInstance.execute({ cmd: 'snapshot' });
      setCurrentPage(browserEngineInstance.currentPage);
      setCurrentHTML(browserEngineInstance.currentHTML);
      setLastResult(snapResult);

      const discResult = await browserEngineInstance.execute({ cmd: 'discover' });
      if (discResult.actions) {
        setActions(discResult.actions);
      }
    };
    init();
  }, []);

  const handleLayerChange = (layer: 1 | 2 | 3) => {
    setActiveLayer(layer);
    browserEngineInstance.setActiveLayer(layer);
    if (currentPage) {
      browserEngineInstance.execute({ cmd: layer === 3 ? 'render' : 'snapshot' }).then((res) => {
        setCurrentPage(browserEngineInstance.currentPage);
        setLastResult(res);
      });
    }
  };

  const handleSelectSample = async (sample: SamplePageDefinition) => {
    setActiveSampleId(sample.id);
    browserEngineInstance.loadSample(sample, activeLayer === 3);
    setCurrentPage(browserEngineInstance.currentPage);
    setCurrentHTML(browserEngineInstance.currentHTML);

    const discRes = await browserEngineInstance.execute({ cmd: 'discover' });
    if (discRes.actions) {
      setActions(discRes.actions);
    }
    setLastResult(discRes);
    setSelectedAction(null);
    setLastResource(null);
  };

  const handleExecuteCommand = async (cmd: Command): Promise<Result> => {
    const res = await browserEngineInstance.execute(cmd);
    setLastResult(res);

    if (browserEngineInstance.currentPage) {
      setCurrentPage(browserEngineInstance.currentPage);
      setCurrentHTML(browserEngineInstance.currentHTML);
    }

    if (res.actions) {
      setActions(res.actions);
      setActiveTab('actions');
    }

    if (res.action) {
      setSelectedAction(res.action);
      setActiveTab('actions');
    }

    if (res.resource) {
      setLastResource(res.resource);
      setActiveTab('actions');
    }

    if (res.matches) {
      setLastMatches(res.matches);
      setActiveTab('dom');
    }

    if (cmd.cmd === 'markdown' || cmd.cmd === 'text') {
      setActiveTab('markdown');
    }

    return res;
  };

  const handleInspectAction = async (ref: string) => {
    const res = await handleExecuteCommand({ cmd: 'inspect', ref });
    if (res.action) {
      setSelectedAction(res.action);
    }
  };

  const handleRunAction = async (ref: string, confirm = true) => {
    const res = await handleExecuteCommand({ cmd: 'run', ref, confirm });
    if (res.resource) {
      setLastResource(res.resource);
    }
    return res.resource;
  };

  const handleQuerySelector = async (selector: string): Promise<ElementMatch[]> => {
    const res = await handleExecuteCommand({ cmd: 'query', selector });
    return res.matches || [];
  };

  const currentMemory = lastResult?.memory || { rss_mib: 7.52, heap_mib: 0.46 };
  const currentDuration = lastResult?.duration_ms ?? 5;

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col selection:bg-cyan-500/20 selection:text-cyan-300">
      <Header
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        activeLayer={activeLayer}
        setActiveLayer={handleLayerChange}
        memory={currentMemory}
        durationMs={currentDuration}
      />

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 space-y-4">
        {activeTab === 'browser' && (
          <PageViewer
            page={currentPage}
            currentHTML={currentHTML}
            onExecute={handleExecuteCommand}
            onSelectSample={handleSelectSample}
            activeSampleId={activeSampleId}
          />
        )}

        {activeTab === 'realsites' && (
          <RealSitesTestRunner />
        )}

        {activeTab === 'actions' && (
          <ActionsTable
            actions={actions}
            onInspect={handleInspectAction}
            onRunAction={handleRunAction}
            selectedAction={selectedAction}
            lastResource={lastResource}
          />
        )}

        {activeTab === 'dom' && (
          <DOMInspector
            page={currentPage}
            onQuery={handleQuerySelector}
            lastMatches={lastMatches}
          />
        )}

        {activeTab === 'markdown' && (
          <MarkdownViewer page={currentPage} />
        )}

        {activeTab === 'benchmark' && (
          <TokenBenchmark />
        )}

        {activeTab === 'arena' && (
          <ArenaServicePanel />
        )}
      </main>

      {/* Agent Terminal Docked at Bottom */}
      <CommandConsole onExecute={handleExecuteCommand} lastResult={lastResult} />
    </div>
  );
}

export default App;
