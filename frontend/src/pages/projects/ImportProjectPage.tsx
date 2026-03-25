import type { Component } from 'solid-js';
import PageContainer from '../../components/layout/PageContainer';
import ImportWizard from '../../components/import/ImportWizard';

const ImportProjectPage: Component = () => {
  return (
    <PageContainer
      title="Import Project"
      description="Import a project from a Git repository, provider, or archive"
    >
      <ImportWizard />
    </PageContainer>
  );
};

export default ImportProjectPage;
