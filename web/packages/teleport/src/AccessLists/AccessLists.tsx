/**
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import React, { useEffect, useState } from 'react';

import { Alert, Box, ButtonPrimary, Flex, Link, Text } from 'design';

import {
  FeatureBox,
  FeatureHeader,
  FeatureHeaderTitle,
} from 'teleport/components/Layout';
import ResourceEditor from 'teleport/components/ResourceEditor';
import useResources from 'teleport/components/useResources';
import useTeleport from 'teleport/useTeleport';
import { CaptureEvent, userEventService } from 'teleport/services/userEvent';

import { useServerSidePagination } from 'teleport/components/hooks';

import { AccessListList } from './AccessListList';
import DeleteAccessList from './DeleteAccessList';
import { useAccessLists, State } from './useAccessLists';

import templates from './templates';

export function AccessListsContainer() {
  const ctx = useTeleport();
  const state = useAccessLists(ctx);
  return <AccessLists {...state} />;
}

export function AccessLists(props: State) {
  const { remove, save, fetch } = props;
  const [search, setSearch] = useState('');

  const serverSidePagination = useServerSidePagination({
    pageSize: 20,
    fetchFunc: async (_, params) => {
      const { items, startKey } = await fetch(params);
      return { agents: items as any[], startKey };
    },
    clusterId: '',
    params: { search },
  });
  const { modifyFetchedData } = serverSidePagination;

  const resources = useResources(
    serverSidePagination.fetchedData.agents,
    templates
  );
  const title =
    resources.status === 'creating'
      ? 'Create a new access list'
      : 'Edit access list';

  async function handleSave(content: string): Promise<void> {
    const name = resources.item.name;
    const isNew = resources.status === 'creating';
    const response = await save(name, content, isNew);
    modifyFetchedData(p => {
      const index = p.agents.findIndex((a: any) => a.id === response.id);
      if (index >= 0) {
        const newAgents = [...p.agents];
        newAgents[index] = response;
        return {
          ...p,
          agents: newAgents,
        };
      } else {
        return {
          ...p,
          agents: [response, ...p.agents],
        };
      }
    });
  }

  useEffect(() => {
    serverSidePagination.fetch();
  }, [search]);

  const handleCreate = () => {
    resources.create('access_list');

    userEventService.captureUserEvent({
      event: CaptureEvent.CreateNewRoleClickEvent,
    });
  };

  async function handleDelete(): Promise<void> {
    await remove(resources.item.name);
    modifyFetchedData(p => ({
      ...p,
      agents: p.agents.filter((r: any) => r.id !== resources.item.id),
    }));
  }

  return (
    <FeatureBox>
      <FeatureHeader alignItems="center">
        <FeatureHeaderTitle>Access Lists</FeatureHeaderTitle>
        <ButtonPrimary ml="auto" width="240px" onClick={handleCreate}>
          CREATE NEW ACCESS LIST
        </ButtonPrimary>
      </FeatureHeader>
      {serverSidePagination.attempt.status === 'failed' && (
        <Alert>{serverSidePagination.attempt.statusText}</Alert>
      )}
      <Flex>
        <Box width="100%" mr="6" mb="4">
          <AccessListList
            serversidePagination={serverSidePagination}
            onSearchChange={setSearch}
            search={search}
            onEdit={resources.edit}
            onDelete={resources.remove}
          />
        </Box>
        <Box
          ml="auto"
          width="240px"
          color="text.main"
          style={{ flexShrink: 0 }}
        >
          <Text typography="h6" mb={3} caps>
            Access List management
          </Text>
          <Text typography="subtitle1" mb={3}>
            Access Lists allow Teleport to grant SSH, database, or application
            access to external users and systems. They can be used to
            delegate access to users outside of your organization.
          </Text>
          <Text>
            Learn more in{' '}
            <Link
              color="text.main"
              target="_blank"
              href="https://goteleport.com/docs/access-guides/access-lists/"
            >
              the Access Lists documentation
            </Link>{' '}
            section of online documentation.
          </Text>
        </Box>
      </Flex>
      {(resources.status === 'creating' || resources.status === 'editing') && (
        <ResourceEditor
          docsURL="https://goteleport.com/docs/access-guides/access-lists/"
          title={title}
          text={resources.item.content}
          name={resources.item.name}
          isNew={resources.status === 'creating'}
          onSave={handleSave}
          onClose={resources.disregard}
          directions={<Directions />}
          kind={resources.item.kind}
        />
      )}
      {resources.status === 'removing' && (
        <DeleteAccessList
          name={resources.item.name}
          onClose={resources.disregard}
          onDelete={handleDelete}
        />
      )}
    </FeatureBox>
  );
}

function Directions() {
  return (
    <>
      Access Lists are defined using{' '}
      <Link
        color="text.main"
        target="_blank"
        href="https://en.wikipedia.org/wiki/YAML"
      >
        YAML format
      </Link>
      . YAML is sensitive to white space, so please be careful.
    </>
  );
}
